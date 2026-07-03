package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound indicates the key is absent from the cache.
var ErrNotFound = errors.New("cache: key not found")

// Redis is a Redis-backed cache store.
type Redis struct {
	client *redis.Client
}

// Open connects to Redis, verifies the connection with PING, and returns a
// Redis cache plus a close function for shutdown.
func Open(ctx context.Context, url string) (*Redis, func() error, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Redis{client: client}, client.Close, nil
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