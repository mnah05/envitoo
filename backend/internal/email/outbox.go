package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/helper"
)

// countPendingOutboxEmails is a lightweight, non-locking count used to decide
// whether there is any work to do before starting a transaction.
const countPendingOutboxEmails = `SELECT COUNT(*) FROM email_outbox WHERE status = 'PENDING' AND retries < $1`

// OutboxConfig holds the tunables for the outbox drainer.
type OutboxConfig struct {
	BatchSize        int
	MaxRetries       int
	AckWaitLimit     time.Duration
	RetentionPeriod  time.Duration
	CleanupBatchSize int
}

// Drainer implements the transactional outbox pattern: it reads PENDING
// rows from the email_outbox table, publishes them to NATS JetStream, and
// records the outcome (PUBLISHED or retry/FAILED). It also deletes stale
// outbox rows so the table does not grow indefinitely.
type Drainer struct {
	pool *pgxpool.Pool
	js   jetstream.JetStream
	cfg  OutboxConfig
	log  *slog.Logger
}

// NewDrainer builds the outbox drainer. js must be the same JetStream
// context the EMAILS stream was ensured on.
func NewDrainer(pool *pgxpool.Pool, js jetstream.JetStream, cfg OutboxConfig, log *slog.Logger) *Drainer {
	return &Drainer{pool: pool, js: js, cfg: cfg, log: log}
}

// errAckTimeout is returned when an async publish ack does not arrive before
// the wait deadline. Kept as a sentinel so callers can compare / use Error().
var errAckTimeout = errors.New("ack timeout")

// resolvePubAck waits for a JetStream PubAckFuture result. Uses a two-phase
// select to avoid the Go runtime's random-select pitfall: when both ctx.Done
// and Ok/Err channels are ready simultaneously, a naive select may choose
// ctx.Done and report a false timeout. After ctx fires we try Ok/Err once
// more in a non-blocking select before giving up.
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

// PublishBatch implements one drain cycle of the transactional outbox:
// 1. SELECT a batch of PENDING rows with SKIP LOCKED (avoids worker contention).
// 2. Publish each row to JetStream (async, deduplicated by msg ID).
// 3. Resolve publish acks and split into succeeded/failed.
// 4. UPDATE succeeded rows to PUBLISHED, failed rows to retry+1 (or FAILED).
// All DB writes run in a single transaction so partial failures do not
// lose rows or double-publish.
func (d *Drainer) PublishBatch(ctx context.Context) error {
	// Check for pending rows before acquiring a transaction. This avoids
	// opening and immediately rolling back a transaction on every poll when
	// the outbox is empty, which is the common case.
	var pendingCount int
	if err := d.pool.QueryRow(ctx, countPendingOutboxEmails, int32(d.cfg.MaxRetries)).Scan(&pendingCount); err != nil {
		return fmt.Errorf("count pending outbox emails: %w", err)
	}
	if pendingCount == 0 {
		return nil
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer helper.RollbackTx(ctx, tx, d.log) // no-op once committed

	queries := db.New(tx)

	// 1. Grab a batch of pending rows. SKIP LOCKED means concurrent workers
	//    won't fight over the same rows.
	batch, err := queries.FetchPendingOutboxEmails(ctx, db.FetchPendingOutboxEmailsParams{
		Limit:   int32(d.cfg.BatchSize),
		Retries: int32(d.cfg.MaxRetries),
	})
	if err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}

	d.log.Info("outbox_batch_fetched", "count", len(batch))

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
		payload, err := json.Marshal(EmailMessage{
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
			subject = DefaultTopicFor(row.Template)
		}

		future, err := d.js.PublishAsync(subject, payload,
			jetstream.WithMsgID(row.ID.String()))
		if err != nil {
			failed = append(failed, failure{id: row.ID, retries: row.Retries, err: err.Error(), topic: subject})
			continue
		}
		futures = append(futures, pending{row: row, future: future, topic: subject})
	}

	// 3. Resolve acks and split into succeeded / failed.
	ctxAck, cancel := context.WithTimeout(ctx, d.cfg.AckWaitLimit)
	defer cancel()

	for _, p := range futures {
		if err := resolvePubAck(ctxAck, p.future.Ok(), p.future.Err()); err != nil {
			failed = append(failed, failure{id: p.row.ID, retries: p.row.Retries, err: err.Error(), topic: p.topic})
			continue
		}
		succeeded = append(succeeded, p.row.ID)
		d.log.Info("outbox_publish_ok",
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
		permanentlyFailed := f.retries+1 >= int32(d.cfg.MaxRetries)
		if permanentlyFailed {
			d.log.Error("outbox_publish_permanent_fail",
				"outbox_id", f.id,
				"topic", f.topic,
				"attempts", d.cfg.MaxRetries,
				"error", f.err,
			)
		} else {
			d.log.Warn("outbox_publish_failed",
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
			Retries:   int32(d.cfg.MaxRetries),
		}); err != nil {
			return err
		}
	}

	d.log.Info("outbox_batch_done", "succeeded", len(succeeded), "failed", len(failed))
	return tx.Commit(ctx)
}

// DeleteStaleBatch performs a single batch deletion of outbox rows that
// have been PUBLISHED or FAILED for longer than the retention period. The
// batch size is capped by cfg.CleanupBatchSize to keep each deletion fast
// and avoid long-held locks on the outbox table.
func (d *Drainer) DeleteStaleBatch(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-d.cfg.RetentionPeriod)

	queries := db.New(d.pool)
	return queries.DeleteStaleOutboxEmails(ctx, db.DeleteStaleOutboxEmailsParams{
		PublishedAt: pgtype.Timestamp{Time: cutoff, Valid: true},
		Limit:       int32(d.cfg.CleanupBatchSize),
	})
}
