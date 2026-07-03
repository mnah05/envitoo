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