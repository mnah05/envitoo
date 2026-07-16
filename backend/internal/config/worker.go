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

func intOr(k *koanf.Koanf, key string, def int) int {
	if !k.Exists(key) {
		return def
	}
	return k.Int(key)
}
