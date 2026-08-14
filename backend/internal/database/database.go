package database

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds the tunables for a pgx connection pool. It is intentionally
// independent of config.Config so it can be reused by the API server, the
// email outbox worker, and any other process that needs a Postgres pool.
type PoolConfig struct {
	MaxConns             int32
	MinConns             int32
	MaxConnLifetime      time.Duration
	MaxConnIdleTime      time.Duration
	HealthCheckPeriod    time.Duration
	ApplicationName      string
	DisablePreparedStmts bool
	// RequireTLS rejects connection strings that disable TLS when true.
	RequireTLS bool
}

// DefaultPoolConfig returns sensible defaults for a web/worker workload.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          20,
		MinConns:          2,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
		ApplicationName:   "envitoo-backend",
	}
}

// Database wraps a pgx connection pool.
//
// Note 1: A connection pool manages multiple database connections so the
// application can handle concurrent requests without opening a new TCP
// connection for every operation. pgxpool is the pool implementation for
// the pgx driver, which is the most popular Postgres driver for Go.
type Database struct {
	pool *pgxpool.Pool
	cfg  PoolConfig
}

// tlsDisabled checks whether a PostgreSQL connection string explicitly
// disables TLS. It handles both URL-style (?sslmode=disable) and keyword/
// DSN-style (sslmode=disable) connection strings.
func tlsDisabled(conn string) bool {
	u, err := url.Parse(conn)
	if err == nil {
		mode := strings.ToLower(u.Query().Get("sslmode"))
		if mode == "disable" {
			return true
		}
	}
	return strings.Contains(strings.ToLower(conn), "sslmode=disable")
}

// Open parses the database URL from cfg, configures the connection pool, and
// verifies connectivity with a Ping before returning.
//
// Note 2: The pool is configured fully before creating it because pgxpool
// applies its settings (like MaxConns) at pool creation time. Changing them
// after NewWithConfig would have no effect.
func Open(ctx context.Context, url string, cfg PoolConfig) (*Database, error) {
	if cfg.RequireTLS && tlsDisabled(url) {
		return nil, fmt.Errorf("database url disables TLS but TLS is required")
	}

	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolCfg.ConnConfig.RuntimeParams["application_name"] = cfg.ApplicationName

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

	warnOnPoolOversubscription(ctx, pool, cfg.MaxConns)

	return &Database{pool: pool, cfg: cfg}, nil
}

// warnOnPoolOversubscription compares the pool's MaxConns against the
// database's max_connections and logs a warning when this single process
// could consume a large share of the server's connection budget. The check
// is advisory only: hard limits are the database's business. Multiple
// processes (api + worker + mailer) share the budget, so a warning here is
// not necessarily a misconfiguration — it means headroom must be planned.
func warnOnPoolOversubscription(ctx context.Context, pool *pgxpool.Pool, maxConns int32) {
	if maxConns <= 0 {
		return
	}
	var dbMaxConns int
	if err := pool.QueryRow(ctx, "SHOW max_connections").Scan(&dbMaxConns); err != nil {
		slog.Warn("could not read max_connections", "error", err)
		return
	}
	if dbMaxConns <= 0 {
		return
	}
	share := float64(maxConns) / float64(dbMaxConns)
	if share >= 0.8 {
		slog.Warn("pool MaxConns consumes a large share of database max_connections",
			"max_conns", maxConns, "db_max_connections", dbMaxConns, "share", share)
	}
}

// Pool returns the underlying pgxpool.Pool so callers can use pgx-native
// query methods like pool.Query, pool.Exec, or pool.Begin.
func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

// Config returns the pool configuration used to create this Database.
func (d *Database) Config() PoolConfig {
	return d.cfg
}

// Ping checks that the database is reachable. This is used by health check
// endpoints to report the database status without exposing the pool directly.
func (d *Database) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Stat returns current pool statistics. Useful for health/metrics endpoints.
func (d *Database) Stat() *pgxpool.Stat {
	return d.pool.Stat()
}

// Close shuts down the connection pool, releasing all connections back to the
// database. Should be called during graceful shutdown.
func (d *Database) Close() {
	d.pool.Close()
}
