package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/config"
	"github.com/mnah05/envitoo/internal/db"
)

// runCleanup periodically deletes email outbox rows that have been
// PUBLISHED or FAILED for longer than cfg.RetentionPeriod. This prevents
// the outbox table from growing indefinitely. Runs every
// cfg.CleanupInterval on a ticker.
func runCleanup(ctx context.Context, pool *pgxpool.Pool, cfg config.WorkerConfig, log *slog.Logger) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := deleteStaleBatch(ctx, pool, cfg)
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

// deleteStaleBatch performs a single batch deletion of stale outbox rows.
// The batch size is capped by cfg.CleanupBatchSize to keep each deletion
// fast and avoid long-held locks on the outbox table.
func deleteStaleBatch(ctx context.Context, pool *pgxpool.Pool, cfg config.WorkerConfig) (int64, error) {
	cutoff := time.Now().Add(-cfg.RetentionPeriod)

	queries := db.New(pool)
	return queries.DeleteStaleOutboxEmails(ctx, db.DeleteStaleOutboxEmailsParams{
		PublishedAt: pgtype.Timestamp{Time: cutoff, Valid: true},
		Limit:       int32(cfg.CleanupBatchSize),
	})
}
