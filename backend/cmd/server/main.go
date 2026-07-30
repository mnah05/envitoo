package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/config"
	"github.com/mnah05/envitoo/internal/database"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/logging"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/server"
)

// main is the API server entry point. It initialises the database pool,
// Redis cache (if configured), Clerk SDK, and the Echo HTTP server, then
// blocks on SIGINT/SIGTERM for graceful shutdown.
func main() {
	log := logging.New("api")
	slog.SetDefault(log)

	_ = godotenv.Load()
	_ = godotenv.Overload(".env.local")

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	clerk.SetKey(cfg.ClerkSecretKey)

	// Pre-warm the Clerk JWKS cache at startup so the first authenticated
	// request doesn't pay the ~1s latency of fetching JWKS from Clerk's API.
	jwksClient := middleware.InitJWKSClient(context.Background())
	if jwksClient != nil {
		log.Info("jwks cache pre-warmed")
	}

	// signal.NotifyContext creates a context that is cancelled when the
	// process receives SIGINT (SIGTERM) (Docker stop / kill).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := database.Open(ctx, cfg)
	if err != nil {
		log.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// sqlc-generated queries need a *pgxpool.Pool (or a tx wrapper).
	// db.New creates the query struct; each method call maps to a SQL query.
	queries := db.New(database.Pool())
	if _, err := queries.Test(ctx); err != nil {
		log.Error("database test query failed", "error", err)
		os.Exit(1)
	}

	log.Info("database connection pool ready",
		"max_conns", cfg.DBMaxConns,
		"min_conns", cfg.DBMinConns,
	)

	// Redis is optional. When unavailable, cache.GetJSON falls through to
	// the fallback function on every call (no caching).
	var appCache *cache.Store
	if cfg.RedisURL != "" {
		redis, closeRedis, err := cache.Open(ctx, cfg.RedisURL)
		if err != nil {
			log.Warn("redis unavailable, caching disabled", "error", err)
		} else {
			defer func() {
				if err := closeRedis(); err != nil {
					log.Error("failed to close redis", "error", err)
				}
			}()
			appCache = cache.New(redis, log)
			redis.StartHealthCheck(ctx, log, 10*time.Second)
			log.Info("redis cache active")
		}
	}

	e := echo.New()
	e.HideBanner = false
	e.HidePort = false
	server.New(database.Pool(), queries, appCache, cfg.OTPSecret, cfg.HealthzToken, log, jwksClient).Register(ctx, e)

	// Start the HTTP server in a goroutine so we can listen for shutdown
	// signals concurrently. The server is configured with explicit timeouts
	// to prevent slow-client attacks and resource leaks.
	serveErr := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", ":8080")
		serveErr <- e.StartServer(&http.Server{
			Addr:         ":8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		})
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
		}
	}

	// Graceful shutdown: stop accepting new requests and wait up to 10s
	// for in-flight requests to complete before force-closing.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
}
