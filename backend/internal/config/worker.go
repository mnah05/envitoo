package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/nats-io/nats.go"
)

// WorkerConfig holds configuration for the email outbox drainer worker.
//
// Note 1: The email outbox pattern uses the database as a queue table and NATS
// as a notification channel. This config controls how aggressively the worker
// polls, retries, and cleans up old records.
type WorkerConfig struct {
	DatabaseURL      string
	NATSURL          string
	BatchSize        int
	PollInterval     time.Duration
	AckWaitLimit     time.Duration
	MaxRetries       int
	CleanupInterval  time.Duration
	RetentionPeriod  time.Duration
	CleanupBatchSize int
}

// LoadWorker reads environment variables for the outbox drainer worker.
//
// Note 2: Unlike Load() for the API server, this function uses the WORKER_
// prefix instead of DB_/CLERK_. This keeps the worker's env namespace
// separate so the same machine can run both processes with different .env
// files.
//
// Note 3: NATS_URL defaults to nats.DefaultURL ("nats://localhost:4222")
// which matches the local docker-compose setup. In production this would be
// overridden to point at a NATS cluster.
func LoadWorker() (WorkerConfig, error) {
	k := koanf.New(".")

	err := k.Load(env.Provider("", ".", func(key string) string {
		if strings.HasPrefix(key, "WORKER_") || key == "DB_URL" || key == "NATS_URL" {
			return key
		}
		return ""
	}), nil)
	if err != nil {
		return WorkerConfig{}, fmt.Errorf("load worker env config: %w", err)
	}

	dbURL := k.String("DB_URL")
	if dbURL == "" {
		return WorkerConfig{}, fmt.Errorf("DB_URL is required")
	}

	natsURL := k.String("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	return WorkerConfig{
		DatabaseURL:      dbURL,
		NATSURL:          natsURL,
		BatchSize:        intOr(k, "WORKER_BATCH_SIZE", 50),
		PollInterval:     durationOr(k, "WORKER_POLL_INTERVAL", 2*time.Second),
		AckWaitLimit:     durationOr(k, "WORKER_ACK_WAIT", 30*time.Second),
		MaxRetries:       intOr(k, "WORKER_MAX_RETRIES", 5),
		CleanupInterval:  durationOr(k, "WORKER_CLEANUP_INTERVAL", time.Hour),
		RetentionPeriod:  durationOr(k, "WORKER_RETENTION_PERIOD", 14*24*time.Hour),
		CleanupBatchSize: intOr(k, "WORKER_CLEANUP_BATCH_SIZE", 1000),
	}, nil
}

// intOr is a config helper similar to int32Or in server.go — returns the env
// var if set, otherwise the provided default. The int type is used here
// (instead of int32) because batch sizes and counts are small enough that
// the wider type is unnecessary.
//
// Note 4: Both intOr and durationOr are unexported duplicates of the helpers
// in server.go. In a larger project these would live in a shared helper
// package, but keeping them local avoids cross-package dependencies for
// something this small.
func intOr(k *koanf.Koanf, key string, def int) int {
	if !k.Exists(key) {
		return def
	}
	return k.Int(key)
}
