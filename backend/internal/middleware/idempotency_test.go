package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"github.com/mnah05/envitoo/internal/cache"
)

func openTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	r, closeRedis, err := cache.Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		mr.Close()
		t.Fatalf("open redis: %v", err)
	}
	_ = slog.Default()
	return r.Client(), func() {
		closeRedis()
		mr.Close()
	}
}

func setupIdemEcho(rdb *redis.Client) *echo.Echo {
	e := echo.New()
	e.Use(Idempotency(rdb))
	e.POST("/test", func(c echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{"ok": "true"})
	})
	return e
}

func postWithKey(e *echo.Echo, key string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestIdempotency_NilRedis_PassThrough(t *testing.T) {
	e := setupIdemEcho(nil)
	rec := postWithKey(e, "key-1", `{"x":1}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestIdempotency_NoKeyHeader_PassThrough(t *testing.T) {
	rdb, cleanup := openTestRedis(t)
	defer cleanup()
	e := setupIdemEcho(rdb)

	rec := postWithKey(e, "", `{"x":1}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 without header, got %d", rec.Code)
	}
}

func TestIdempotency_FirstRequest_Succeeds(t *testing.T) {
	rdb, cleanup := openTestRedis(t)
	defer cleanup()
	e := setupIdemEcho(rdb)

	rec := postWithKey(e, "key-1", `{"x":1}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["ok"] != "true" {
		t.Fatalf("expected ok=true, got %v", body)
	}
	if rec.Header().Get("Idempotent-Replayed") != "" {
		t.Fatal("should not have Idempotent-Replayed on first request")
	}
}

func TestIdempotency_SamePayload_ReplaysResponse(t *testing.T) {
	rdb, cleanup := openTestRedis(t)
	defer cleanup()
	e := setupIdemEcho(rdb)

	// First request
	rec1 := postWithKey(e, "key-1", `{"x":1}`)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first request failed: %d", rec1.Code)
	}

	// Second request with same payload should replay
	rec2 := postWithKey(e, "key-1", `{"x":1}`)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected replayed 201, got %d", rec2.Code)
	}
	if rec2.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("expected Idempotent-Replayed: true on replay")
	}
	var body map[string]string
	json.NewDecoder(rec2.Body).Decode(&body)
	if body["ok"] != "true" {
		t.Fatalf("replayed body mismatch: %v", body)
	}
}

func TestIdempotency_DifferentPayload_ReturnsConflict(t *testing.T) {
	rdb, cleanup := openTestRedis(t)
	defer cleanup()
	e := setupIdemEcho(rdb)

	// First request with payload A
	rec1 := postWithKey(e, "key-1", `{"x":1}`)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first request failed: %d", rec1.Code)
	}

	// Second request with same key but different payload
	rec2 := postWithKey(e, "key-1", `{"x":2}`)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec2.Code)
	}
}

func TestIdempotency_ConcurrentRequest_ReturnsConflict(t *testing.T) {
	rdb, cleanup := openTestRedis(t)
	defer cleanup()

	body := `{"x":1}`
	payloadHash := hashSHA256([]byte(body))

	// Simulate in_progress by injecting directly into Redis
	key := "idem:POST:/test::conc-key"
	doc := `{"status":"in_progress","payload_hash":"` + payloadHash + `"}`
	if err := rdb.Set(context.Background(), key, doc, 0).Err(); err != nil {
		t.Fatalf("redis set failed: %v", err)
	}

	e := setupIdemEcho(rdb)
	rec := postWithKey(e, "conc-key", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for in_progress, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "2" {
		t.Fatalf("expected Retry-After: 2, got %q", rec.Header().Get("Retry-After"))
	}
}
