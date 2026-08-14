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
	"github.com/mnah05/envitoo/internal/database"
	"github.com/mnah05/envitoo/internal/email"
	"github.com/mnah05/envitoo/internal/logging"
)

// main is the outbox drainer worker entry point. On a poll interval it
// reads PENDING rows from the email_outbox table, publishes them to NATS
// JetStream, and records the outcome (PUBLISHED or retry/FAILED). It also
// runs a background cleanup goroutine that deletes stale outbox rows.
// The batch logic itself lives in email.Drainer.
func main() {
	log := logging.New("worker")
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()
	_ = godotenv.Overload(".env.local")

	cfg, err := config.LoadWorker()
	if err != nil {
		log.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(ctx, cfg.DatabaseURL, database.PoolConfig{
		MaxConns:             cfg.DBMaxConns,
		MinConns:             cfg.DBMinConns,
		MaxConnLifetime:      cfg.DBMaxConnLifetime,
		MaxConnIdleTime:      cfg.DBMaxConnIdleTime,
		HealthCheckPeriod:    cfg.DBHealthCheckPeriod,
		ApplicationName:      "envitoo-worker",
		DisablePreparedStmts: cfg.DisablePreparedStmts,
		RequireTLS:           cfg.DBRequireTLS,
	})
	if err != nil {
		log.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	pool := db.Pool()

	nc, err := nats.Connect(cfg.NATSURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
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

	for attempt := range 10 {
		_, err = email.EnsureEmailsStream(ctx, js)
		if err == nil {
			break
		}
		log.Warn("emails_stream_retry", "attempt", attempt+1, "error", err)
		select {
		case <-ctx.Done():
			log.Error("ensure EMAILS stream cancelled", "error", ctx.Err())
			os.Exit(1)
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	if err != nil {
		log.Error("ensure EMAILS stream", "error", err)
		os.Exit(1)
	}

	drainer := email.NewDrainer(pool, js, email.OutboxConfig{
		BatchSize:        cfg.BatchSize,
		MaxRetries:       cfg.MaxRetries,
		AckWaitLimit:     cfg.AckWaitLimit,
		RetentionPeriod:  cfg.RetentionPeriod,
		CleanupBatchSize: cfg.CleanupBatchSize,
	}, log)

	go runCleanup(ctx, drainer, cfg, log)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	log.Info("outbox_drainer_ready", "nats", cfg.NATSURL, "batch_size", cfg.BatchSize)

	for {
		select {
		case <-ctx.Done():
			log.Info("worker_shutdown")
			return
		case <-ticker.C:
			if err := drainer.PublishBatch(ctx); err != nil {
				log.Error("publish_batch_failed", "error", err)
			}
		}
	}
}

// runCleanup periodically deletes email outbox rows that have been
// PUBLISHED or FAILED for longer than cfg.RetentionPeriod. This prevents
// the outbox table from growing indefinitely. Runs every
// cfg.CleanupInterval on a ticker.
func runCleanup(ctx context.Context, drainer *email.Drainer, cfg config.WorkerConfig, log *slog.Logger) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := drainer.DeleteStaleBatch(ctx)
			if err != nil {
				log.Error("outbox_cleanup_failed", "error", err)
				continue
			}
			if n > 0 {
				log.Info("outbox_cleanup_ok", "deleted", n)
			}
		}
	}
}
