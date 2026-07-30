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
//
// Note 1: A context logger is preferred over a global logger because each
// request may carry its own attributes (request_id, user_id) that would be
// lost with a global. The request-scoped logger is built in the middleware
// and propagated down through the context.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, log)
}

// LoggerFromContext returns the logger stored in ctx by WithLogger, or
// slog.Default() if none was stored. This ensures logging never panics
// even if the injection middleware did not run.
//
// Note 2: The fallback-to-default pattern means middleware that runs before
// RequestLogger (like the panic recovery middleware) still has a usable
// logger without explicit ordering dependencies.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// RequestLogger attaches a request-scoped logger (with request_id) to the
// context and emits one structured http_request access log after the
// handler returns. Place after Recover and RequestID. Skips GET /healthz.
//
// Note 3: The access log is emitted AFTER the handler runs (in the
// middleware's "return" path), so it has the final status code and response
// size. This is the standard Echo middleware pattern — before the handler
// runs setup, after the handler runs teardown.
//
// Note 4: The log level varies by status code: Info for 2xx/3xx, Warn for
// 4xx client errors, Error for 5xx server errors. This lets operators set
// alert thresholds on the Error level without noise from 404s.
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

// skipAccessLog returns true for health check pings, which are frequent and
// uninteresting. Every request to /healthz would otherwise fill the log with
// noise that has no diagnostic value.
func skipAccessLog(c echo.Context) bool {
	return c.Request().Method == "GET" && c.Request().URL.Path == "/healthz"
}
