package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
)

func doHealthRequest(s *Server, query string) *httptest.ResponseRecorder {
	e := echo.New()
	e.GET("/healthz", s.Health)
	url := "/healthz"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// openTestCache starts an in-memory miniredis and wraps it in a cache.Store.
// The returned close function stops the server; the store keeps its handle.
func openTestCache(t *testing.T) (*cache.Store, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	redis, closeRedis, err := cache.Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		mr.Close()
		t.Fatalf("open redis: %v", err)
	}
	store := cache.New(redis, slog.Default())
	return store, func() {
		_ = closeRedis()
		mr.Close()
	}
}

func TestHealth_RequiresToken(t *testing.T) {
	s := &Server{healthToken: "open2005", log: slog.Default()}

	if rec := doHealthRequest(s, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: status = %d, want 401", rec.Code)
	}
	if rec := doHealthRequest(s, "pass=wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: status = %d, want 401", rec.Code)
	}
}

func TestHealth_NoPool(t *testing.T) {
	s := &Server{healthToken: "open2005", log: slog.Default()}
	rec := doHealthRequest(s, "pass=open2005")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status = %v, want ok", payload["status"])
	}
	if payload["db"] != "n/a" {
		t.Fatalf("db = %v, want n/a", payload["db"])
	}
	if payload["redis"] != "n/a" {
		t.Fatalf("redis = %v, want n/a", payload["redis"])
	}
	if payload["service"] != "envitoo-api" {
		t.Fatalf("service = %v, want envitoo-api", payload["service"])
	}
	if _, ok := payload["uptime_seconds"]; !ok {
		t.Fatal("uptime_seconds missing")
	}
}

func TestHealth_DBUp(t *testing.T) {
	pool, _ := openServerTestDB(t)
	s := &Server{pool: pool, healthToken: "", log: slog.Default()}

	rec := doHealthRequest(s, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "ok" || payload["db"] != "ok" {
		t.Fatalf("payload = %v, want status ok + db ok", payload)
	}
	if payload["redis"] != "n/a" {
		t.Fatalf("redis = %v, want n/a when cache absent", payload["redis"])
	}
}

func TestHealth_RedisUp(t *testing.T) {
	pool, _ := openServerTestDB(t)
	store, closeRedis := openTestCache(t)
	defer closeRedis()
	s := &Server{pool: pool, cache: store, log: slog.Default()}

	rec := doHealthRequest(s, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "ok" || payload["db"] != "ok" || payload["redis"] != "ok" {
		t.Fatalf("payload = %v, want all ok", payload)
	}
}

func TestHealth_RedisDown(t *testing.T) {
	pool, _ := openServerTestDB(t)
	store, closeRedis := openTestCache(t)
	closeRedis() // close the backing redis; ping should now fail
	s := &Server{pool: pool, cache: store, log: slog.Default()}

	rec := doHealthRequest(s, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (redis is optional); body: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status = %v, want ok despite redis down", payload["status"])
	}
	if payload["db"] != "ok" {
		t.Fatalf("db = %v, want ok", payload["db"])
	}
	if payload["redis"] != "unreachable" {
		t.Fatalf("redis = %v, want unreachable", payload["redis"])
	}
}
