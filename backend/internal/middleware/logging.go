package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

type loggerContextKey struct{}

// WithLogger returns a new context that carries the given logger so
// downstream middleware and handlers can retrieve it via
// LoggerFromContext. This is the slog context-logger pattern implemented
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

// RequestLogger attaches a request-scoped logger (with request_id) to the
// context and emits one structured http_request access log after the
// handler returns. Place after Recover and RequestID. Skips GET /healthz.
func RequestLogger(base *slog.Logger) echo.MiddlewareFunc {
	if base == nil {
		base = slog.Default()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			if rid == "" {
				rid = c.Request().Header.Get(echo.HeaderXRequestID)
			}

			log := base
			if rid != "" {
				log = log.With("request_id", rid)
			}
			ctx := WithLogger(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			if skipAccessLog(c) {
				return err
			}

			status := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok && he.Code > 0 {
					status = he.Code
				} else if status < 400 {
					status = 500
				}
			}
			if status == 0 {
				status = 200
			}

			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			attrs := []any{
				"method", c.Request().Method,
				"path", path,
				"status", status,
				"latency_ms", time.Since(start).Milliseconds(),
				"bytes_out", c.Response().Size,
			}
			if rid != "" {
				attrs = append(attrs, "request_id", rid)
			}
			if u, ok := UserFromContext(c); ok && u != nil {
				attrs = append(attrs, "user_id", u.ID)
			}

			switch {
			case status >= 500:
				log.Error("http_request", attrs...)
			case status >= 400:
				log.Warn("http_request", attrs...)
			default:
				log.Info("http_request", attrs...)
			}

			return err
		}
	}
}

func skipAccessLog(c echo.Context) bool {
	return c.Request().Method == "GET" && c.Request().URL.Path == "/healthz"
}
