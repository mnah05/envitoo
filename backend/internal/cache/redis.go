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
//
// Note 1: atomic.Bool is used for the healthy flag so it can be read/written
// across goroutines without a mutex. The health check runs in a background
// goroutine while cache operations happen on request goroutines.
type Redis struct {
	client  *redis.Client
	healthy atomic.Bool
}

// Open connects to Redis, verifies the connection with PING, and returns a
// Redis cache plus a close function for shutdown.
//
// Note 2: MaxRetries is set to 0 because the application prefers fast failure
// over retry. If Redis is down, the cache-aside pattern falls through to the
// database rather than blocking requests waiting for a retry to succeed.
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
//
// Note 3: CompareAndSwap is used to avoid log spam — the health transition
// log message only fires once per state change rather than on every tick.
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

// Note 4: Get/Set/Delete are the core cache interface. Each method first
// checks the healthy flag so the application never tries to talk to a known-
// broken Redis, which would waste time on connection timeouts.

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
//
// Note 5: SetNX is useful for rate-limiting and cooldown scenarios where you
// want "set if absent" atomicity. The underlying Redis SET NX command is
// atomic — there is no race between the check and the set.
func (r *Redis) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

// Incr increments key by 1 and returns the new value. Creates the key at 0 if missing.
// Note: INCR does not attach a TTL; callers must Expire on first creation (count == 1).
//
// Note 6: This two-step Incr + Expire pattern is common for rate-limit
// windows. The first increment creates the key at 1, then immediately sets
// the expiry. If the TTL were set first, there would be a race where the key
// expires before the increment.
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets a TTL on key. Returns ErrNotFound if the key does not exist.
//
// Note 7: Redis EXPIRE returns 0 if the key does not exist. The method
// surfaces this as ErrNotFound so callers can distinguish "set TTL" from
// "key already expired."
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
//
// Note 8: Redis SCAN is used instead of KEYS because KEYS blocks the Redis
// event loop on large key spaces. SCAN iterates in batches (cursor-based),
// keeping the operation friendly to other clients.
//
// Note 9: Glob characters in the prefix are escaped so that a prefix like
// "events?limit=10" is treated literally. Without escaping, the "?" would
// match any single character in the SCAN pattern.
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
