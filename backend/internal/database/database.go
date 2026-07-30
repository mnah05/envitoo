package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/config"
)

// Database wraps a pgx connection pool.
//
// Note 1: A connection pool manages multiple database connections so the
// application can handle concurrent requests without opening a new TCP
// connection for every operation. pgxpool is the pool implementation for
// the pgx driver, which is the most popular Postgres driver for Go.
type Database struct {
	pool *pgxpool.Pool
}

// Open parses the database URL from cfg, configures the connection pool, and
// verifies connectivity with a Ping before returning.
//
// Note 2: The pool is configured fully before creating it because pgxpool
// applies its settings (like MaxConns) at pool creation time. Changing them
// after NewWithConfig would have no effect.
//
// Note 3: Setting application_name helps identify this application's
// connections in Postgres pg_stat_activity and slow-query logs, which is
// very useful when multiple services talk to the same database.
func Open(ctx context.Context, cfg config.Config) (*Database, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MinConns = cfg.DBMinConns
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.DBHealthCheckPeriod
	poolCfg.ConnConfig.RuntimeParams["application_name"] = "envitoo-backend"

	// DisablePreparedStmts is useful behind PgBouncer in transaction mode,
	// which does not support prepared statements across connections. When
	// enabled, the driver uses the simple query protocol (Exec) instead.
	if cfg.DisablePreparedStmts {
		poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Ping before returning ensures the application fails early if the
	// database is unreachable, rather than failing on the first request.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Database{pool: pool}, nil
}

// Pool returns the underlying pgxpool.Pool so callers can use pgx-native
// query methods like pool.Query, pool.Exec, or pool.Begin.
func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

// Ping checks that the database is reachable. This is used by health check
// endpoints to report the database status without exposing the pool directly.
func (d *Database) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Close shuts down the connection pool, releasing all connections back to the
// database. Should be called during graceful shutdown.
func (d *Database) Close() {
	d.pool.Close()
}
