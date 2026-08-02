package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
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
}
