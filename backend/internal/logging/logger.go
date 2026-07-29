// Package logging provides a shared slog factory for API, worker, and mailer
// processes. JSON is the default format so logs are machine-parseable in prod;
// set LOG_FORMAT=text for local readability.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a process logger with a stable "service" attribute.
//
// Environment:
//   - LOG_FORMAT: "json" (default) or "text"
//   - LOG_LEVEL:  "debug", "info" (default), "warn", "error"
func New(service string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler).With("service", service)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
