package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

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

	ok, err := c.SetNX(ctx, "cooldown", []byte("1"), time.Minute)
	if err != nil || !ok {
		t.Fatalf("setnx first: ok=%v err=%v", ok, err)
	}
	ok, err = c.SetNX(ctx, "cooldown", []byte("1"), time.Minute)
	if err != nil || ok {
		t.Fatalf("setnx second: ok=%v err=%v", ok, err)
	}

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
	if err := c.DeleteByPrefix(ctx, "envito:v1:events:public:"); err != nil {
		t.Fatalf("delete by prefix: %v", err)
	}
	if _, err := c.Get(ctx, "envito:v1:events:public:1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected public list key deleted")
	}
	if _, err := c.Get(ctx, "envito:v1:other"); err != nil {
		t.Fatalf("expected unrelated key to remain: %v", err)
	}
}
