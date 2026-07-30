package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/mnah05/envitoo/internal/db"
)

func TestRequestLogger_EmitsAccessLogWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	e := echo.New()
	e.Use(echoMiddleware.RequestID())
	e.Use(RequestLogger(base))
	e.GET("/api/events", func(c echo.Context) error {
		log := LoggerFromContext(c.Request().Context())
		log.Info("handler_ran")
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req.Header.Set(echo.HeaderXRequestID, "req-test-123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get(echo.HeaderXRequestID); got != "req-test-123" {
		t.Fatalf("response request id=%q", got)
	}

	// The log buffer should contain both the handler log and the access log
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected handler + access log lines, got %q", buf.String())
	}

	var access map[string]any
	foundAccess := false
	for _, line := range lines {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("json line: %v %q", err, line)
		}
		if rec["msg"] == "http_request" {
			access = rec
			foundAccess = true
		}
		if rid, ok := rec["request_id"].(string); ok && rid != "req-test-123" {
			t.Fatalf("unexpected request_id=%v in %v", rid, rec["msg"])
		}
	}
	if !foundAccess {
		t.Fatalf("missing http_request log in %q", buf.String())
	}
	if access["method"] != "GET" {
		t.Fatalf("method=%v", access["method"])
	}
	if access["path"] != "/api/events" {
		t.Fatalf("path=%v", access["path"])
	}
	if int(access["status"].(float64)) != http.StatusNoContent {
		t.Fatalf("status=%v", access["status"])
	}
}

func TestRequestLogger_SkipsHealthz(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	e := echo.New()
	e.Use(echoMiddleware.RequestID())
	e.Use(RequestLogger(base))
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Healthcheck should NOT produce an access log entry
	if strings.Contains(buf.String(), "http_request") {
		t.Fatalf("healthz should not emit access log, got %q", buf.String())
	}
}

func TestRequestLogger_IncludesUserIDWhenPresent(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	uid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	e := echo.New()
	e.Use(echoMiddleware.RequestID())
	e.Use(RequestLogger(base))
	e.GET("/api/me", func(c echo.Context) error {
		c.Set(userContextKey, &db.User{ID: uid, Email: "x@example.com"})
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var access map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec map[string]any
		_ = json.Unmarshal([]byte(line), &rec)
		if rec["msg"] == "http_request" {
			access = rec
		}
	}
	if access == nil {
		t.Fatalf("missing access log: %q", buf.String())
	}
	if access["user_id"] != uid.String() {
		t.Fatalf("user_id=%v want %s", access["user_id"], uid)
	}
}

func TestLoggerFromContext_Default(t *testing.T) {
	if LoggerFromContext(t.Context()) == nil {
		t.Fatal("expected default logger")
	}
}
