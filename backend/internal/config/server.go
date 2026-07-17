package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabaseURL          string
	DBMaxConns           int32
	DBMinConns           int32
	DBMaxConnLifetime    time.Duration
	DBMaxConnIdleTime    time.Duration
	DBHealthCheckPeriod  time.Duration
	DisablePreparedStmts bool
	ClerkSecretKey       string
	RedisURL             string
	// OTPSecret is the HMAC key for guest email OTP. Optional at process
	// start; OTP endpoints return 503 if empty.
	OTPSecret string
}

// Load reads environment variables prefixed with DB_ and CLERK_ into a Config.
func Load() (Config, error) {
	k := koanf.New(".")

	err := k.Load(env.Provider("", ".", func(key string) string {
		if strings.HasPrefix(key, "DB_") || strings.HasPrefix(key, "CLERK_") || key == "REDIS_URL" || key == "OTP_SECRET" {
			return key
		}
		return ""
	}), nil)
	if err != nil {
		return Config{}, fmt.Errorf("load env config: %w", err)
	}

	dbURL := k.String("DB_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DB_URL is required")
	}
	clerkSecretKey := k.String("CLERK_SECRET_KEY")
	if clerkSecretKey == "" {
		return Config{}, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	return Config{
		DatabaseURL:          dbURL,
		DBMaxConns:           int32Or(k, "DB_MAX_CONNS", 40),
		DBMinConns:           int32Or(k, "DB_MIN_CONNS", 2),
		DBMaxConnLifetime:    durationOr(k, "DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime:    durationOr(k, "DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		DBHealthCheckPeriod:  durationOr(k, "DB_HEALTH_CHECK_PERIOD", time.Minute),
		DisablePreparedStmts: k.Bool("DB_DISABLE_PREPARED_STATEMENTS"),
		ClerkSecretKey:       clerkSecretKey,
		RedisURL:             k.String("REDIS_URL"),
		OTPSecret:            k.String("OTP_SECRET"),
	}, nil
}

func int32Or(k *koanf.Koanf, key string, def int32) int32 {
	if !k.Exists(key) {
		return def
	}
	return int32(k.Int64(key))
}

func durationOr(k *koanf.Koanf, key string, def time.Duration) time.Duration {
	if !k.Exists(key) {
		return def
	}
	return k.Duration(key)
}
