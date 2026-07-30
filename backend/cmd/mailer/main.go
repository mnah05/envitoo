package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mnah05/envitoo/internal/config"
	"github.com/mnah05/envitoo/internal/email"
	"github.com/mnah05/envitoo/internal/logging"
)

// main is the mailer worker entry point. It subscribes to the EMAILS
// JetStream consumer, renders HTML emails from templates, and sends them
// via the Resend API. Each email is processed in a per-message handler
// with a 45-second timeout.
func main() {
	log := logging.New("mailer")
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()
	_ = godotenv.Overload(".env.local")

	cfg, err := config.LoadMailer()
	if err != nil {
		log.Error("load config", "error", err)
		os.Exit(1)
	}

	renderer, err := email.NewRenderer(cfg.AppPublicURL)
	if err != nil {
		log.Error("init renderer", "error", err)
		os.Exit(1)
	}

	sender, err := email.NewSender(cfg.ResendAPIKey, cfg.EmailFrom)
	if err != nil {
		log.Error("init sender", "error", err)
		os.Exit(1)
	}

	// NATS connection with infinite reconnect. The DisconnectErrHandler
	// and ReconnectHandler log state transitions so operators can observe
	// transient broker issues.
	nc, err := nats.Connect(cfg.NATSURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Warn("nats_disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Info("nats_reconnected")
		}),
	)
	if err != nil {
		log.Error("nats connect", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Error("jetstream init", "error", err)
		os.Exit(1)
	}

	var consumer jetstream.Consumer
	for attempt := 0; attempt < 10; attempt++ {
		consumer, err = email.EnsureMailerConsumer(ctx, js)
		if err == nil {
			break
		}
		log.Warn("mailer_consumer_retry", "attempt", attempt+1, "error", err)
		select {
		case <-ctx.Done():
			log.Error("ensure mailer consumer cancelled", "error", ctx.Err())
			os.Exit(1)
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	if err != nil {
		log.Error("ensure mailer consumer", "error", err)
		os.Exit(1)
	}

	log.Info("mailer_ready",
		"stream", email.StreamName,
		"consumer", email.ConsumerName,
	)

	consCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		handleMsg(ctx, msg, renderer, sender, log)
	})
	if err != nil {
		log.Error("consume", "error", err)
		os.Exit(1)
	}
	defer consCtx.Stop()

	<-ctx.Done()
	log.Info("mailer_shutdown")
}

func handleMsg(ctx context.Context, msg jetstream.Msg, renderer *email.Renderer, sender *email.Sender, log *slog.Logger) {
	// handleMsg processes a single email message from the JetStream consumer.
	// Bound each send with a 45s timeout so a hung Resend call cannot block
	// the ack deadline forever.
	sendCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var payload email.EmailMessage
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		log.Error("mailer_invalid_payload", "subject", msg.Subject(), "error", err)
		// Poison message — acknowledge so it does not retry forever.
		_ = msg.Ack()
		return
	}

	msgLog := log.With(
		"outbox_id", payload.OutboxID,
		"template", payload.Template,
		"topic", msg.Subject(),
	)

	rendered, err := renderer.Render(payload.Template, payload.TemplateData)
	if err != nil {
		msgLog.Error("mailer_render_failed", "error", err)
		if isPermanentRenderErr(err) {
			_ = msg.Ack()
			return
		}
		_ = msg.Nak()
		return
	}

	err = sender.Send(sendCtx, email.SendRequest{
		To:             payload.Recipient,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: payload.OutboxID.String(),
	})
	if err != nil {
		msgLog.Error("mailer_send_failed", "error", err)
		if isPermanentSendErr(err) {
			_ = msg.Term()
			return
		}
		_ = msg.Nak()
		return
	}

	if err := msg.Ack(); err != nil {
		msgLog.Error("mailer_ack_failed", "error", err)
		return
	}
	msgLog.Info("mailer_send_ok")
}

// isPermanentRenderErr returns true for template errors that will never
// succeed on retry (e.g. unknown template name, decode failure). The
// message is Ack'd (not Nak'd) so it is removed from the stream.
func isPermanentRenderErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown email template") ||
		strings.Contains(msg, "decode ")
}

// isPermanentSendErr returns true for Resend API errors that indicate a
// bad request (4xx) rather than a transient failure (5xx/timeout).
// Permanent errors call msg.Term() to move them to the terminal backlog;
// transient errors call msg.Nak() for redelivery.
func isPermanentSendErr(err error) bool {
	if err == nil {
		return false
	}
	// Resend returns API errors with status embedded in the message.
	// Treat clear client errors as terminal; everything else is retryable.
	msg := strings.ToLower(err.Error())
	if errors.Is(err, context.Canceled) {
		return false
	}
	permanentHints := []string{
		"status code: 400",
		"status code: 401",
		"status code: 403",
		"status code: 404",
		"status code: 422",
		"invalid api key",
		"validation_error",
	}
	for _, h := range permanentHints {
		if strings.Contains(msg, h) {
			return true
		}
	}
	return false
}
