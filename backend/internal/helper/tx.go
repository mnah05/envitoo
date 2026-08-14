package helper

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// RollbackTx rolls back a transaction from a deferred call, ignoring the
// "already closed" error that pgx returns when the transaction was already
// committed (or rolled back). Real rollback failures are logged so a
// transaction that silently stayed open never goes unnoticed.
//
// Usage:
//
//	tx, err := pool.Begin(ctx)
//	if err != nil { ... }
//	defer helper.RollbackTx(ctx, tx, log)
func RollbackTx(ctx context.Context, tx pgx.Tx, log *slog.Logger) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		log.Warn("tx rollback failed", "error", err)
	}
}
