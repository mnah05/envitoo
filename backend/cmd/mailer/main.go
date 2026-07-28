package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mnah05/envitoo/internal/config"
	"github.com/mnah05/envitoo/internal/email"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	cfg, err := config.LoadMailer()
	if err != nil {
		log.Fatal(err)
	}

	renderer, err := email.NewRenderer(cfg.AppPublicURL)
	if err != nil {
		log.Fatal(err)
	}

	sender, err := email.NewSender(cfg.ResendAPIKey, cfg.EmailFrom)
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(cfg.NATSURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Printf("nats disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Printf("nats reconnected")
		}),
	)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("jetstream init: %v", err)
	}

	consumer, err := email.EnsureMailerConsumer(ctx, js)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("mailer ready: stream=%s consumer=%s from=%s", email.StreamName, email.ConsumerName, cfg.EmailFrom)

	consCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		handleMsg(ctx, msg, renderer, sender)
	})
	if err != nil {
		log.Fatalf("consume: %v", err)
	}
	defer consCtx.Stop()

	<-ctx.Done()
	log.Println("shutting down mailer")
}

func handleMsg(ctx context.Context, msg jetstream.Msg, renderer *email.Renderer, sender *email.Sender) {
	// Bound each send so a hung Resend call cannot block AckWait forever.
	sendCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var payload email.EmailMessage
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		log.Printf("mailer: invalid payload subject=%s: %v", msg.Subject(), err)
		// Poison message — acknowledge so it does not retry forever.
		_ = msg.Ack()
		return
	}

	rendered, err := renderer.Render(payload.Template, payload.TemplateData)
	if err != nil {
		log.Printf("mailer: render failed outbox_id=%s template=%s: %v", payload.OutboxID, payload.Template, err)
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
		log.Printf("mailer: send failed outbox_id=%s recipient=%s: %v", payload.OutboxID, redactEmail(payload.Recipient), err)
		if isPermanentSendErr(err) {
			_ = msg.Term()
			return
		}
		_ = msg.Nak()
		return
	}

	if err := msg.Ack(); err != nil {
		log.Printf("mailer: ack failed outbox_id=%s: %v", payload.OutboxID, err)
	}
}

func isPermanentRenderErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown email template") ||
		strings.Contains(msg, "decode ")
}

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

func redactEmail(addr string) string {
	addr = strings.TrimSpace(addr)
	at := strings.LastIndex(addr, "@")
	if at <= 1 {
		return "***"
	}
	return addr[:1] + "***" + addr[at:]
}
