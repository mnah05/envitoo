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
)

type EmailOutboxRow struct {
	ID           uuid.UUID
	Recipient    string
	Template     string
	TemplateData json.RawMessage
	Topic        string
	Retries      int
}

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

	// 1. Grab a batch of pending rows. SKIP LOCKED means concurrent worker
	rows, err := tx.Query(ctx, `
		SELECT id, recipient, template, template_data, topic, retries
		FROM email_outbox
		WHERE status = 'PENDING'
		  AND retries < $2
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, cfg.BatchSize, cfg.MaxRetries)
	if err != nil {
		return err
	}

	var batch []EmailOutboxRow
	for rows.Next() {
		var r EmailOutboxRow
		if err := rows.Scan(&r.ID, &r.Recipient, &r.Template, &r.TemplateData, &r.Topic, &r.Retries); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
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
		retries int
		err     string
	}
	type pending struct {
		row    EmailOutboxRow
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

		// Use the stored topic. Fall back to the template->topic map only
		// if a row somehow has an empty/unrecognized topic — this should
		// be rare if every INSERT path sets it correctly.
		subject := row.Topic
		if subject == "" {
			subject = defaultTopicFor(row.Template)
		}

		// Dedupe key: if this exact outbox row gets published twice
		// (e.g. retried after a crash before status was updated),
		// JetStream will drop the duplicate within its dedupe window.
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
		_, err = tx.Exec(ctx, `
			UPDATE email_outbox
			SET status = 'PUBLISHED', published_at = now()
			WHERE id = ANY($1)
		`, succeeded)
		if err != nil {
			return err
		}
	}

	// 5. Bump retries + record last_error for failures. Rows stay PENDING
	//    and are retried until retries reaches maxRetries, then FAILED.
	for _, f := range failed {
		permanentlyFailed := f.retries+1 >= cfg.MaxRetries
		if permanentlyFailed {
			log.Printf("publish permanently failed for %s after %d attempts: %s", f.id, cfg.MaxRetries, f.err)
		} else {
			log.Printf("publish failed for %s: %s", f.id, f.err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE email_outbox
			SET retries = retries + 1,
			    last_error = $2,
			    status = CASE
			        WHEN retries + 1 >= $3 THEN 'FAILED'::email_outbox_status
			        ELSE status
			    END,
			    failed_at = CASE
			        WHEN retries + 1 >= $3 THEN now()
			        ELSE failed_at
			    END
			WHERE id = $1
		`, f.id, f.err, cfg.MaxRetries)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
