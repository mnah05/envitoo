package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/config"
)

func runCleanup(ctx context.Context, pool *pgxpool.Pool, cfg config.WorkerConfig) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := deleteStaleBatch(ctx, pool, cfg)
			if err != nil {
				log.Printf("cleanup: %v", err)
				continue
			}
			if n > 0 {
				log.Printf("cleanup: deleted %d stale outbox rows", n)
			}
		}
	}
}

func deleteStaleBatch(ctx context.Context, pool *pgxpool.Pool, cfg config.WorkerConfig) (int64, error) {
	cutoff := time.Now().Add(-cfg.RetentionPeriod)

	tag, err := pool.Exec(ctx, `
		DELETE FROM email_outbox
		WHERE id IN (
			SELECT id
			FROM email_outbox
			WHERE (
				status = 'PUBLISHED'
				AND published_at < $1
			) OR (
				status = 'FAILED'
				AND COALESCE(failed_at, created_at) < $1
			)
			LIMIT $2
		)
	`, cutoff, cfg.CleanupBatchSize)
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}
