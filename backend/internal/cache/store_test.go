package cache

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func openTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	redis, closeRedis, err := Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		mr.Close()
		t.Fatalf("open redis: %v", err)
	}
	store := New(redis, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return store, func() {
		closeRedis()
		mr.Close()
	}
}

func TestGetJSON_CacheHit(t *testing.T) {
	ctx := context.Background()
	store, cleanup := openTestStore(t)
	defer cleanup()

	var loads atomic.Int32
	load := func(context.Context) (string, error) {
		loads.Add(1)
		return "cached-value", nil
	}

	v1, err := GetJSON(ctx, store, "test:key", time.Minute, load)
	if err != nil || v1 != "cached-value" {
		t.Fatalf("first read: %q err=%v", v1, err)
	}
	v2, err := GetJSON(ctx, store, "test:key", time.Minute, load)
	if err != nil || v2 != "cached-value" {
		t.Fatalf("second read: %q err=%v", v2, err)
	}
	if loads.Load() != 1 {
		t.Fatalf("loader called %d times, want 1", loads.Load())
	}
}

func TestGetJSON_NilStoreBypassesCache(t *testing.T) {
	ctx := context.Background()
	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		loads.Add(1)
		return 1, nil
	}
	_, _ = GetJSON(ctx, nil, "x", time.Minute, load)
	_, _ = GetJSON(ctx, nil, "x", time.Minute, load)
	if loads.Load() != 2 {
		t.Fatalf("loader called %d times, want 2", loads.Load())
	}
}

func TestGetSetDelete(t *testing.T) {
	ctx := context.Background()
	store, cleanup := openTestStore(t)
	defer cleanup()

	key := "GET:/api/events/123"
	data := []byte(`{"id":"123"}`)

	if err := store.Set(ctx, key, data, time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("got %q, want %q", got, data)
	}

	store.Delete(ctx, key)
	if _, err := store.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatal("expected key deleted")
	}
}

func TestNilStoreIsNoop(t *testing.T) {
	ctx := context.Background()
	var nilStore *Store

	if _, err := nilStore.Get(ctx, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatal("expected ErrNotFound from nil store")
	}
	if err := nilStore.Set(ctx, "x", []byte("v"), time.Minute); err != nil {
		t.Fatalf("set on nil store: %v", err)
	}
	nilStore.Delete(ctx, "x")
	nilStore.DeletePrefix(ctx, "x")
}

func TestDeletePrefix(t *testing.T) {
	ctx := context.Background()
	store, cleanup := openTestStore(t)
	defer cleanup()

	store.Set(ctx, "GET:/api/events?limit=20", []byte("a"), time.Minute)
	store.Set(ctx, "GET:/api/events?limit=10", []byte("b"), time.Minute)
	store.Set(ctx, "GET:/api/events/123", []byte("c"), time.Minute)

	store.DeletePrefix(ctx, "GET:/api/events?")

	if _, err := store.Get(ctx, "GET:/api/events?limit=20"); !errors.Is(err, ErrNotFound) {
		t.Fatal("expected list key deleted")
	}
	if _, err := store.Get(ctx, "GET:/api/events?limit=10"); !errors.Is(err, ErrNotFound) {
		t.Fatal("expected other list key deleted")
	}
	if _, err := store.Get(ctx, "GET:/api/events/123"); err != nil {
		t.Fatal("expected single event key to remain")
	}
}
