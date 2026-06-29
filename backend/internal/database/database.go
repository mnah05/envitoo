package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/config"
)

// Database wraps a pgx connection pool.
type Database struct {
	pool *pgxpool.Pool
}

// Open parses the database URL from cfg, configures the connection pool, and
// verifies connectivity with a Ping before returning.
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

	if cfg.DisablePreparedStmts {
		poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Database{pool: pool}, nil
}

// Pool returns the underlying pgxpool.Pool.
func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

// Ping checks that the database is reachable.
func (d *Database) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Close shuts down the connection pool.
func (d *Database) Close() {
	d.pool.Close()
}
