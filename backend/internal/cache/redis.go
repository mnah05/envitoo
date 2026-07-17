package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound indicates the key is absent from the cache.
var ErrNotFound = errors.New("cache: key not found")

// ErrUnavailable indicates Redis is required but missing or unhealthy.
var ErrUnavailable = errors.New("cache: redis unavailable")

// Redis is a Redis-backed cache store with health tracking.
type Redis struct {
	client  *redis.Client
	healthy atomic.Bool
}

// Open connects to Redis, verifies the connection with PING, and returns a
// Redis cache plus a close function for shutdown.
func Open(ctx context.Context, url string) (*Redis, func() error, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, nil, fmt.Errorf("parse redis url: %w", err)
	}
	opts.MaxRetries = 0

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("redis ping: %w", err)
	}

	r := &Redis{client: client}
	r.healthy.Store(true)
	return r, client.Close, nil
}

// IsHealthy reports whether Redis is believed to be reachable.
func (r *Redis) IsHealthy() bool {
	return r.healthy.Load()
}

// StartHealthCheck launches a background goroutine that pings Redis every
// interval and updates the healthy flag. The goroutine exits when ctx is
// cancelled. Only one health-check goroutine should run per Redis instance.
func (r *Redis) StartHealthCheck(ctx context.Context, log *slog.Logger, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := r.client.Ping(ctx).Err()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					if r.healthy.CompareAndSwap(true, false) {
						log.Warn("redis unhealthy, caching disabled")
					}
				} else {
					if r.healthy.CompareAndSwap(false, true) {
						log.Info("redis healthy, caching resumed")
					}
				}
			}
		}
	}()
}

func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// SetNX sets key only if it does not already exist. Returns true if the key was set.
func (r *Redis) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

// Incr increments key by 1 and returns the new value. Creates the key at 0 if missing.
// Note: INCR does not attach a TTL; callers must Expire on first creation (count == 1).
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets a TTL on key. Returns ErrNotFound if the key does not exist.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	ok, err := r.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// DeleteByPrefix removes every key starting with prefix.
// The prefix is treated as a literal string (glob chars are escaped).
func (r *Redis) DeleteByPrefix(ctx context.Context, prefix string) error {
	escaped := strings.NewReplacer(
		`\`, `\\`,
		`?`, `\?`,
		`*`, `\*`,
		`[`, `\[`,
		`]`, `\]`,
	).Replace(prefix)

	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, escaped+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
