package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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
// via the Resend API. Each email is processed by email.Consumer with a
// 45-second timeout.
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
	for attempt := range 10 {
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

	mailer := email.NewConsumer(renderer, sender)

	consCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		mailer.Handle(ctx, msg, log)
	})
	if err != nil {
		log.Error("consume", "error", err)
		os.Exit(1)
	}
	defer consCtx.Stop()

	<-ctx.Done()
	log.Info("mailer_shutdown")
}
