package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	EventTTL              = 5 * time.Minute
	PublicListTTL         = 2 * time.Minute
	GuestTicketTTL        = 2 * time.Minute
	VolunteerTTL          = 5 * time.Minute
	MyRegistrationsTTL    = 2 * time.Minute
	MyVolunteerAppsTTL    = 2 * time.Minute
	EventRegistrationsTTL = 2 * time.Minute
	MyEventStaffTTL       = 5 * time.Minute
)

// Store wraps Redis with cache-aside reads and delete helpers.
// A nil Store (or no Redis) means caching is off — every call becomes a no-op
// or falls through to the database.
//
// Note 1: The nil-safety pattern (checking s == nil || s.redis == nil at the
// start of every method) means callers do not need a nil check before calling
// cache operations. Passing a nil *Store is safe and simply bypasses caching.
//
// Note 2: singleflight.Group coalesces concurrent GetJSON calls for the same
// key — only one goroutine runs the load function; the rest wait for its
// result. This prevents a thundering herd when a popular cache key expires.
type Store struct {
	redis *Redis
	log   *slog.Logger
	group singleflight.Group
}

func New(redis *Redis, log *slog.Logger) *Store {
	if log == nil {
		log = slog.Default()
	}
	return &Store{redis: redis, log: log}
}

// Note 3: The "soft" methods (Get, Set, Delete) treat Redis errors as soft
// failures — they return ErrNotFound for reads and silently ignore write
// errors. This keeps the application running when Redis is down rather than
// failing every request. The "hard" methods (GetBytes, SetBytes, etc.) use
// requireRedis() to return ErrUnavailable when Redis is required.

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	if s == nil || s.redis == nil {
		return nil, ErrNotFound
	}
	if !s.redis.IsHealthy() {
		return nil, ErrNotFound
	}
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return nil, ErrNotFound
	}
	return data, nil
}

func (s *Store) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if s == nil || s.redis == nil {
		return nil
	}
	if !s.redis.IsHealthy() {
		return nil
	}
	if err := s.redis.Set(ctx, key, data, ttl); err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, keys ...string) {
	if s == nil || s.redis == nil || len(keys) == 0 {
		return
	}
	if !s.redis.IsHealthy() {
		return
	}
	if err := s.redis.Delete(ctx, keys...); err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		} else {
			s.log.WarnContext(ctx, "cache delete failed", "keys", keys, "error", err)
		}
	}
}

// Available reports whether Redis is present and believed healthy.
// Use for features that require Redis (OTP), not optional HTTP caching.
func (s *Store) Available() bool {
	return s != nil && s.redis != nil && s.redis.IsHealthy()
}

func (s *Store) requireRedis() error {
	if !s.Available() {
		return ErrUnavailable
	}
	return nil
}

// GetBytes returns the value for key. Unlike Get, it returns real Redis errors
// and ErrUnavailable when Redis is down (not a soft miss).
func (s *Store) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if err := s.requireRedis(); err != nil {
		return nil, err
	}
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return nil, err
	}
	return data, nil
}

// SetBytes stores value with TTL. Fails hard if Redis is unavailable.
func (s *Store) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := s.requireRedis(); err != nil {
		return err
	}
	if err := s.redis.Set(ctx, key, value, ttl); err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return err
	}
	return nil
}

// SetNX sets key only if absent. Fails hard if Redis is unavailable.
func (s *Store) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	if err := s.requireRedis(); err != nil {
		return false, err
	}
	ok, err := s.redis.SetNX(ctx, key, value, ttl)
	if err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return false, err
	}
	return ok, nil
}

// DeleteKeys deletes keys and returns any Redis error (unlike soft Delete).
func (s *Store) DeleteKeys(ctx context.Context, keys ...string) error {
	if err := s.requireRedis(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	if err := s.redis.Delete(ctx, keys...); err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return err
	}
	return nil
}

// Incr increments key by 1. Call Expire when the returned count is 1 so the key
// is not permanent (INCR does not set TTL).
func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	if err := s.requireRedis(); err != nil {
		return 0, err
	}
	n, err := s.redis.Incr(ctx, key)
	if err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return 0, err
	}
	return n, nil
}

// Expire sets a TTL on key. Fails hard if Redis is unavailable.
func (s *Store) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := s.requireRedis(); err != nil {
		return err
	}
	if err := s.redis.Expire(ctx, key, ttl); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		}
		return err
	}
	return nil
}

func (s *Store) DeletePrefix(ctx context.Context, prefix string) {
	if s == nil || s.redis == nil || prefix == "" {
		return
	}
	if !s.redis.IsHealthy() {
		return
	}
	if err := s.redis.DeleteByPrefix(ctx, prefix); err != nil {
		if s.redis.healthy.CompareAndSwap(true, false) {
			s.log.WarnContext(ctx, "redis unhealthy, caching disabled", "error", err)
		} else {
			s.log.WarnContext(ctx, "cache prefix delete failed", "prefix", prefix, "error", err)
		}
	}
}

// GetJSON is a cache-aside read: try cache, on miss call load, cache result.
//
// Note 4: This is a generic function (Go 1.18+) parameterized on any type T.
// The JSON serialization is handled transparently — the caller provides a
// strongly-typed load function and gets the same type back.
//
// Note 5: The singleflight.Do call ensures that when multiple requests miss
// cache simultaneously, only one calls load() while the others wait. This
// protects the database from a spike in reads when a popular key expires.
func GetJSON[T any](ctx context.Context, s *Store, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	if s == nil || s.redis == nil || !s.redis.IsHealthy() {
		return load(ctx)
	}

	if data, err := s.Get(ctx, key); err == nil {
		var v T
		if err := json.Unmarshal(data, &v); err != nil {
			s.log.WarnContext(ctx, "cache decode failed", "key", key, "error", err)
		} else {
			return v, nil
		}
	} else if !errors.Is(err, ErrNotFound) {
		s.log.WarnContext(ctx, "cache read failed", "key", key, "error", err)
	}

	result, err, _ := s.group.Do(key, func() (any, error) {
		value, err := load(ctx)
		if err != nil {
			return nil, err
		}

		if data, err := json.Marshal(value); err != nil {
			s.log.WarnContext(ctx, "cache encode failed", "key", key, "error", err)
		} else if err := s.Set(ctx, key, data, ttl); err != nil {
			s.log.WarnContext(ctx, "cache write failed", "key", key, "error", err)
		}
		return value, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}
