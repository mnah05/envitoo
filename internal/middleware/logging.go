package middleware

import (
	"context"
	"log/slog"
)

type loggerContextKey struct{}

// WithLogger returns a new context that carries the given logger so
// downstream middleware and handlers can retrieve it via
// loggerFromContext. This is the slog context-logger pattern implemented
// manually since log/slog does not provide NewContext/FromContext in
// this Go version.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, log)
}

// LoggerFromContext returns the logger stored in ctx by WithLogger, or
// slog.Default() if none was stored. This ensures logging never panics
// even if the injection middleware did not run.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
