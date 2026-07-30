// Package logging provides a shared slog factory for API, worker, and mailer
// processes. JSON is the default format so logs are machine-parseable in prod;
// set LOG_FORMAT=text for local readability.
//
// Note 1: The package uses Go 1.21's structured logging (log/slog) instead of
// third-party libraries like zerolog or logrus. slog is part of the standard
// library, which means zero dependencies and a consistent API across the Go
// ecosystem going forward.
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
//
// Note 2: The "service" attribute is added at construction time so every log
// line from this process is tagged with its role (e.g. "api", "worker",
// "mailer"). When logs from multiple processes are aggregated, this attribute
// makes it easy to filter by service.
//
// Note 3: slog.NewJSONHandler writes structured JSON that log aggregators
// (like Grafana Loki, Datadog, or CloudWatch) can parse automatically. The
// text format is only used for local development where you want human-readable
// output in the terminal.
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

// parseLevel converts the LOG_LEVEL env var string to a slog.Level value.
// Unknown or missing values fall back to LevelInfo, which is the safest
// default — it logs informational messages and above but suppresses debug
// noise.
//
// Note 4: "warning" is accepted as an alias for "warn" because many operators
// are more familiar with the full word from other logging systems.
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
