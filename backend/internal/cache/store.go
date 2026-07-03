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
	EventTTL       = 5 * time.Minute
	PublicListTTL  = 2 * time.Minute
	GuestTicketTTL = 2 * time.Minute
	VolunteerTTL   = 5 * time.Minute
)

// Store wraps Redis with cache-aside reads and delete helpers.
// A nil Store (or no Redis) means caching is off — every call becomes a no-op
// or falls through to the database.
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

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	if s == nil || s.redis == nil {
		return nil, ErrNotFound
	}
	return s.redis.Get(ctx, key)
}

func (s *Store) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Set(ctx, key, data, ttl)
}

func (s *Store) Delete(ctx context.Context, keys ...string) {
	if s == nil || s.redis == nil || len(keys) == 0 {
		return
	}
	if err := s.redis.Delete(ctx, keys...); err != nil {
		s.log.WarnContext(ctx, "cache delete failed", "keys", keys, "error", err)
	}
}

func (s *Store) DeletePrefix(ctx context.Context, prefix string) {
	if s == nil || s.redis == nil || prefix == "" {
		return
	}
	if err := s.redis.DeleteByPrefix(ctx, prefix); err != nil {
		s.log.WarnContext(ctx, "cache prefix delete failed", "prefix", prefix, "error", err)
	}
}

// GetJSON is a cache-aside read: try cache, on miss call load, cache result.
func GetJSON[T any](ctx context.Context, s *Store, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	if s == nil || s.redis == nil {
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
