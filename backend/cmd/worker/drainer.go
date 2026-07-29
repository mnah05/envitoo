package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mnah05/envitoo/internal/config"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/email"
	"github.com/mnah05/envitoo/internal/logging"
)

// errAckTimeout is returned when an async publish ack does not arrive before
// the wait deadline. Kept as a sentinel so callers can compare / use Error().
var errAckTimeout = errors.New("ack timeout")

// resolvePubAck waits for a JetStream PubAckFuture result.
//
// A plain select on Ok/Err/ctx.Done is unsafe: when the deadline fires, Ok
// (or Err) may already be ready too, and Go picks among ready cases at random.
// That can mark a successful publish as timed out. After Done fires we do a
// non-blocking nested select that prefers Ok/Err when either is ready.
func resolvePubAck(ctx context.Context, ok <-chan *jetstream.PubAck, errCh <-chan error) error {
	select {
	case <-ok:
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		select {
		case <-ok:
			return nil
		case err := <-errCh:
			return err
		default:
			return errAckTimeout
		}
	}
}

func main() {
	log := logging.New("worker")
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()
	_ = godotenv.Load(".env.local")

	cfg, err := config.LoadWorker()
	if err != nil {
		log.Error("load config", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("ping database", "error", err)
		os.Exit(1)
	}

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

	if _, err := email.EnsureEmailsStream(ctx, js); err != nil {
		log.Error("ensure EMAILS stream", "error", err)
		os.Exit(1)
	}

	go runCleanup(ctx, pool, cfg, log)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	log.Info("outbox_drainer_ready", "nats", cfg.NATSURL, "batch_size", cfg.BatchSize)

	for {
		select {
		case <-ctx.Done():
			log.Info("worker_shutdown")
			return
		case <-ticker.C:
			if err := publishBatch(ctx, pool, js, cfg, log); err != nil {
				log.Error("publish_batch_failed", "error", err)
			}
		}
	}
}

func publishBatch(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, cfg config.WorkerConfig, log *slog.Logger) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op once committed

	queries := db.New(tx)

	// 1. Grab a batch of pending rows. SKIP LOCKED means concurrent workers
	//    won't fight over the same rows.
	batch, err := queries.FetchPendingOutboxEmails(ctx, db.FetchPendingOutboxEmailsParams{
		Limit:   int32(cfg.BatchSize),
		Retries: int32(cfg.MaxRetries),
	})
	if err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}

	log.Info("outbox_batch_fetched", "count", len(batch))

	// 2. Publish everything async — PublishAsync returns immediately with a
	//    future you resolve later. Per-row failures are recorded and written
	//    back so the rest of the batch can still commit.
	type failure struct {
		id      uuid.UUID
		retries int32
		err     string
		topic   string
	}
	type pending struct {
		row    db.FetchPendingOutboxEmailsRow
		future jetstream.PubAckFuture
		topic  string
	}

	var succeeded []uuid.UUID
	var failed []failure
	futures := make([]pending, 0, len(batch))

	for _, row := range batch {
		payload, err := json.Marshal(email.EmailMessage{
			OutboxID:     row.ID,
			Recipient:    row.Recipient,
			Template:     row.Template,
			TemplateData: row.TemplateData,
		})
		if err != nil {
			failed = append(failed, failure{id: row.ID, retries: row.Retries, err: err.Error(), topic: row.Topic})
			continue
		}

		subject := row.Topic
		if subject == "" {
			subject = email.DefaultTopicFor(row.Template)
		}

		future, err := js.PublishAsync(subject, payload,
			jetstream.WithMsgID(row.ID.String()))
		if err != nil {
			failed = append(failed, failure{id: row.ID, retries: row.Retries, err: err.Error(), topic: subject})
			continue
		}
		futures = append(futures, pending{row: row, future: future, topic: subject})
	}

	// 3. Resolve acks and split into succeeded / failed.
	ctxAck, cancel := context.WithTimeout(ctx, cfg.AckWaitLimit)
	defer cancel()

	for _, p := range futures {
		if err := resolvePubAck(ctxAck, p.future.Ok(), p.future.Err()); err != nil {
			failed = append(failed, failure{id: p.row.ID, retries: p.row.Retries, err: err.Error(), topic: p.topic})
			continue
		}
		succeeded = append(succeeded, p.row.ID)
		log.Info("outbox_publish_ok",
			"outbox_id", p.row.ID,
			"template", p.row.Template,
			"topic", p.topic,
		)
	}

	// 4. Mark successes as PUBLISHED, in one query.
	if len(succeeded) > 0 {
		if err := queries.MarkOutboxEmailsPublished(ctx, succeeded); err != nil {
			return err
		}
	}

	// 5. Bump retries + record last_error for failures. Rows stay PENDING
	//    and are retried until retries reaches maxRetries, then FAILED.
	for _, f := range failed {
		permanentlyFailed := f.retries+1 >= int32(cfg.MaxRetries)
		if permanentlyFailed {
			log.Error("outbox_publish_permanent_fail",
				"outbox_id", f.id,
				"topic", f.topic,
				"attempts", cfg.MaxRetries,
				"error", f.err,
			)
		} else {
			log.Warn("outbox_publish_failed",
				"outbox_id", f.id,
				"topic", f.topic,
				"retries", f.retries+1,
				"error", f.err,
			)
		}

		lastErr := f.err
		if err := queries.IncrementOutboxEmailRetries(ctx, db.IncrementOutboxEmailRetriesParams{
			ID:        f.id,
			LastError: &lastErr,
			Retries:   int32(cfg.MaxRetries),
		}); err != nil {
			return err
		}
	}

	log.Info("outbox_batch_done", "succeeded", len(succeeded), "failed", len(failed))
	return tx.Commit(ctx)
}
