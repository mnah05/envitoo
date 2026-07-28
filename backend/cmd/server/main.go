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
	"github.com/mnah05/envitoo/internal/server"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	clerk.SetKey(cfg.ClerkSecretKey)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := database.Open(ctx, cfg)
	if err != nil {
		log.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	queries := db.New(database.Pool())
	if _, err := queries.Test(ctx); err != nil {
		log.Error("database test query failed", "error", err)
		os.Exit(1)
	}

	log.Info("database connection pool ready",
		"max_conns", cfg.DBMaxConns,
		"min_conns", cfg.DBMinConns,
	)

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
	server.New(database.Pool(), queries, appCache, cfg.OTPSecret, cfg.HealthzToken, log).Register(ctx, e)

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
}
