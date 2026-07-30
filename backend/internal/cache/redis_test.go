package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// Note 1: Tests use miniredis — an in-memory Redis implementation written
// in pure Go. No actual Redis server is needed to run the test suite.

func TestRedis_GetSetDelete(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	c, closeRedis, err := Open(ctx, "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("open redis: %v", err)
	}
	defer closeRedis()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "v" {
		t.Fatalf("got %q, want v", got)
	}

	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := c.Get(ctx, "k"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected miss, got %v", err)
	}
}

func TestRedis_SetNXIncrExpire(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	c, closeRedis, err := Open(ctx, "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("open redis: %v", err)
	}
	defer closeRedis()

	// First SetNX should succeed (key does not exist)
	ok, err := c.SetNX(ctx, "cooldown", []byte("1"), time.Minute)
	if err != nil || !ok {
		t.Fatalf("setnx first: ok=%v err=%v", ok, err)
	}
	// Second SetNX with the same key should return false
	ok, err = c.SetNX(ctx, "cooldown", []byte("1"), time.Minute)
	if err != nil || ok {
		t.Fatalf("setnx second: ok=%v err=%v", ok, err)
	}

	// Incr creates key at 0, increments to 1
	n, err := c.Incr(ctx, "attempts")
	if err != nil || n != 1 {
		t.Fatalf("incr: n=%d err=%v", n, err)
	}
	if err := c.Expire(ctx, "attempts", time.Minute); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if !mr.Exists("attempts") {
		t.Fatal("attempts missing after expire")
	}
	n, err = c.Incr(ctx, "attempts")
	if err != nil || n != 2 {
		t.Fatalf("incr2: n=%d err=%v", n, err)
	}
}

func TestRedis_DeleteByPrefix(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	c, closeRedis, err := Open(ctx, "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("open redis: %v", err)
	}
	defer closeRedis()

	for _, key := range []string{"envito:v1:events:public:1", "envito:v1:events:public:2", "envito:v1:other"} {
		if err := c.Set(ctx, key, []byte("x"), time.Minute); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	// Delete all keys with prefix "envito:v1:events:public:"
	if err := c.DeleteByPrefix(ctx, "envito:v1:events:public:"); err != nil {
		t.Fatalf("delete by prefix: %v", err)
	}
	// Public list keys should be gone
	if _, err := c.Get(ctx, "envito:v1:events:public:1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected public list key deleted")
	}
	// The unrelated key should still exist
	if _, err := c.Get(ctx, "envito:v1:other"); err != nil {
		t.Fatalf("expected unrelated key to remain: %v", err)
	}
}
