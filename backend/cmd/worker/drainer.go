package main

import (
	"context"
	"encoding/json"
	"log"
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
)

// What actually gets published to JetStream. Keep it self-contained so
// the consumer side doesn't need to hit Postgres to know what to send.
type EmailMessage struct {
	OutboxID     uuid.UUID       `json:"outbox_id"`
	Recipient    string          `json:"recipient"`
	Template     string          `json:"template"`
	TemplateData json.RawMessage `json:"template_data"`
}

const (
	TemplateOTP                = "otp"
	TemplateTicketConfirm      = "ticket_confirm"
	TemplateTicketCancel       = "ticket_cancel"
	TemplateTicketStatusChange = "ticket_status_change"
	TemplateApplicationConfirm = "application_confirm"
	TemplateApplicationCancel  = "application_cancel"
	TemplateCheckIn            = "check_in"
	TemplateEventCancellation  = "event_cancellation"
)

var topicForTemplate = map[string]string{
	TemplateOTP:                "emails.otp",
	TemplateTicketConfirm:      "emails.ticket.confirmed",
	TemplateTicketCancel:       "emails.ticket.cancelled",
	TemplateTicketStatusChange: "emails.ticket.status_changed",
	TemplateApplicationConfirm: "emails.application.confirmed",
	TemplateApplicationCancel:  "emails.application.cancelled",
	TemplateCheckIn:            "emails.checkin",
	TemplateEventCancellation:  "emails.event.cancelled",
}

// defaultTopicFor is only a fallback for rows where topic somehow ended up
// empty (bad insert, old row before the column existed, etc.) so the
// publisher doesn't have to special-case them.
func defaultTopicFor(template string) string {
	if t, ok := topicForTemplate[template]; ok {
		return t
	}
	return "emails.unknown"
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	cfg, err := config.LoadWorker()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Unable to connect to database: ", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Ping failed: ", err)
	}

	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("jetstream init: %v", err)
	}

	go runCleanup(ctx, pool, cfg)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			if err := publishBatch(ctx, pool, js, cfg); err != nil {
				log.Printf("publish batch: %v", err)
			}
		}
	}
}

func publishBatch(ctx context.Context, pool *pgxpool.Pool, js jetstream.JetStream, cfg config.WorkerConfig) error {
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

	// 2. Publish everything async — PublishAsync returns immediately with a
	//    future you resolve later. Per-row failures are recorded and written
	//    back so the rest of the batch can still commit.
	type failure struct {
		id      uuid.UUID
		retries int32
		err     string
	}
	type pending struct {
		row    db.FetchPendingOutboxEmailsRow
		future jetstream.PubAckFuture
	}

	var succeeded []uuid.UUID
	var failed []failure
	futures := make([]pending, 0, len(batch))

	for _, row := range batch {
		payload, err := json.Marshal(EmailMessage{
			OutboxID:     row.ID,
			Recipient:    row.Recipient,
			Template:     row.Template,
			TemplateData: row.TemplateData,
		})
		if err != nil {
			failed = append(failed, failure{id: row.ID, retries: row.Retries, err: err.Error()})
			continue
		}

		subject := row.Topic
		if subject == "" {
			subject = defaultTopicFor(row.Template)
		}

		future, err := js.PublishAsync(subject, payload,
			jetstream.WithMsgID(row.ID.String()))
		if err != nil {
			failed = append(failed, failure{id: row.ID, retries: row.Retries, err: err.Error()})
			continue
		}
		futures = append(futures, pending{row: row, future: future})
	}

	// 3. Resolve acks and split into succeeded / failed.

	ctxAck, cancel := context.WithTimeout(ctx, cfg.AckWaitLimit)
	defer cancel()

	for _, p := range futures {
		select {
		case <-p.future.Ok():
			succeeded = append(succeeded, p.row.ID)
		case err := <-p.future.Err():
			failed = append(failed, failure{id: p.row.ID, retries: p.row.Retries, err: err.Error()})
		case <-ctxAck.Done():
			failed = append(failed, failure{id: p.row.ID, retries: p.row.Retries, err: "ack timeout"})
		}
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
			log.Printf("publish permanently failed for %s after %d attempts: %s", f.id, cfg.MaxRetries, f.err)
		} else {
			log.Printf("publish failed for %s: %s", f.id, f.err)
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

	return tx.Commit(ctx)
}
