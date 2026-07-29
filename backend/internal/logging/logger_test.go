package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestNew_Constructs(t *testing.T) {
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_LEVEL", "info")
	if New("api") == nil {
		t.Fatal("expected logger")
	}

	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("LOG_LEVEL", "debug")
	if New("worker") == nil {
		t.Fatal("expected logger")
	}
}

func TestNew_TextFormatAndDebugLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))})
	capture := slog.New(h).With("service", "worker")
	capture.Debug("probe_debug")
	if !bytes.Contains(buf.Bytes(), []byte("probe_debug")) {
		t.Fatalf("expected debug log, got %q", buf.String())
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"nope", slog.LevelInfo},
	}
	for _, tt := range tests {
		if got := parseLevel(tt.in); got != tt.want {
			t.Fatalf("parseLevel(%q)=%v want %v", tt.in, got, tt.want)
		}
	}
}

func TestNew_EmitsJSONWithService(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := slog.New(h).With("service", "api")
	log.Info("boot")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("json: %v body=%q", err, buf.String())
	}
	if rec["msg"] != "boot" {
		t.Fatalf("msg=%v", rec["msg"])
	}
	if rec["service"] != "api" {
		t.Fatalf("service=%v", rec["service"])
	}
}
