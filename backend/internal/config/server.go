package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

// Config holds all application configuration loaded from environment variables.
//
// Note 1: Configuration is loaded from environment variables rather than a
// config file. This follows the twelve-factor app methodology, making the
// application portable across environments (dev, staging, prod) without
// changing code or config files.
type Config struct {
	DatabaseURL          string
	DBMaxConns           int32
	DBMinConns           int32
	DBMaxConnLifetime    time.Duration
	DBMaxConnIdleTime    time.Duration
	DBHealthCheckPeriod  time.Duration
	DisablePreparedStmts bool
	DBRequireTLS         bool
	AppEnv               string
	ClerkSecretKey       string
	RedisURL             string
	// OTPSecret is the HMAC key for guest email OTP. Optional at process
	// start; OTP endpoints return 503 if empty.
	OTPSecret string
	// HealthzToken, when set, must be provided as ?pass= on GET /healthz.
	HealthzToken string
	// R2 configuration for Cloudflare R2 (S3-compatible object storage).
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string
}

// Load reads environment variables prefixed with DB_ and CLERK_ into a Config.
//
// Note 2: The env.Provider callback acts as an allow-list — only environment
// variables matching the listed prefixes/names are picked up. This prevents
// accidentally leaking unrelated env vars into the application config, which
// is a common security footgun with naive os.Getenv approaches.
//
// Note 3: Koanf was chosen over Go's standard os.Getenv because it provides
// a consistent API for config from multiple sources (env, files, flags), with
// built-in parsing of types like time.Duration and int32.
func Load() (Config, error) {
	k := koanf.New(".")

	err := k.Load(env.Provider("", ".", func(key string) string {
		if strings.HasPrefix(key, "DB_") || strings.HasPrefix(key, "CLERK_") || strings.HasPrefix(key, "R2_") || key == "REDIS_URL" || key == "OTP_SECRET" || key == "HEALTHZ_TOKEN" || key == "APP_ENV" {
			return key
		}
		return ""
	}), nil)
	if err != nil {
		return Config{}, fmt.Errorf("load env config: %w", err)
	}

	// Validate required fields at load time so the application fails fast
	// during startup instead of panic-ing later with a confusing nil-pointer
	// or connection error.
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
		DBMaxConns:           int32Or(k, "DB_MAX_CONNS", 20),
		DBMinConns:           int32Or(k, "DB_MIN_CONNS", 2),
		DBMaxConnLifetime:    durationOr(k, "DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime:    durationOr(k, "DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		DBHealthCheckPeriod:  durationOr(k, "DB_HEALTH_CHECK_PERIOD", time.Minute),
		DisablePreparedStmts: k.Bool("DB_DISABLE_PREPARED_STATEMENTS"),
		DBRequireTLS:         k.Bool("DB_REQUIRE_TLS") || isProduction(k.String("APP_ENV")),
		AppEnv:               k.String("APP_ENV"),
		ClerkSecretKey:       clerkSecretKey,
		RedisURL:             k.String("REDIS_URL"),
		OTPSecret:            k.String("OTP_SECRET"),
		HealthzToken:         k.String("HEALTHZ_TOKEN"),
		R2AccountID:          k.String("R2_ACCOUNT_ID"),
		R2AccessKeyID:        k.String("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey:    k.String("R2_SECRET_ACCESS_KEY"),
		R2BucketName:         k.String("R2_BUCKET_NAME"),
		R2PublicURL:          k.String("R2_PUBLIC_URL"),
	}, nil
}

// isProduction returns true when the environment name indicates a production
// deployment. It is used to opt-in to stricter defaults such as requiring TLS.
func isProduction(env string) bool {
	return strings.ToLower(env) == "production" || strings.ToLower(env) == "prod"
}

// int32Or is a helper that returns the env var value if set, or a default
// otherwise. This pattern avoids repetitive if-exists-else blocks in Load().
//
// Note 4: The function is unexported because it is only used within this
// package. Go's convention is that unexported names serve as package-internal
// implementation details.
func int32Or(k *koanf.Koanf, key string, def int32) int32 {
	if !k.Exists(key) {
		return def
	}
	return int32(k.Int64(key))
}

// durationOr works like int32Or but for time.Duration values.
//
// Note 5: Koanf's Duration() automatically parses strings like "30m" or
// "1h30m" via Go's time.ParseDuration, which is much friendlier for operators
// than requiring millisecond integers in environment variables.
func durationOr(k *koanf.Koanf, key string, def time.Duration) time.Duration {
	if !k.Exists(key) {
		return def
	}
	return k.Duration(key)
}
