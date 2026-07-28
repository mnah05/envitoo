package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// defaultMaxVisitors caps distinct rate-limit keys to prevent unbounded map growth.
const defaultMaxVisitors = 100_000

type RateLimitConfig struct {
	Rate            float64
	Burst           int
	CleanupInterval time.Duration
	CleanupIdleTime time.Duration
	// MaxVisitors is the hard cap on tracked keys. Zero uses defaultMaxVisitors.
	MaxVisitors int
}

var (
	DefaultPublicConfig = RateLimitConfig{
		Rate:            10,
		Burst:           20,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
		MaxVisitors:     defaultMaxVisitors,
	}
	// DefaultOTPConfig is intentionally tighter than DefaultPublicConfig —
	// each OTP request can enqueue a real outbound email.
	DefaultOTPConfig = RateLimitConfig{
		Rate:            5.0 / 60.0, // 5 requests per minute
		Burst:           5,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
		MaxVisitors:     defaultMaxVisitors,
	}
	DefaultAuthConfig = RateLimitConfig{
		Rate:            30,
		Burst:           60,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
		MaxVisitors:     defaultMaxVisitors,
	}
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	config   RateLimitConfig
}

func newIPRateLimiter(ctx context.Context, config RateLimitConfig) *ipRateLimiter {
	if config.MaxVisitors <= 0 {
		config.MaxVisitors = defaultMaxVisitors
	}
	l := &ipRateLimiter{
		visitors: make(map[string]*visitor),
		config:   config,
	}
	go l.cleanupLoop(ctx)
	return l
}

func (l *ipRateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.mu.Lock()
			now := time.Now()
			for ip, v := range l.visitors {
				if now.Sub(v.lastSeen) > l.config.CleanupIdleTime {
					delete(l.visitors, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}

func (l *ipRateLimiter) getVisitor(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if v, exists := l.visitors[key]; exists {
		v.lastSeen = time.Now()
		return v.limiter
	}

	// At capacity: do not grow the map; deny new keys until idle entries are cleaned up.
	if len(l.visitors) >= l.config.MaxVisitors {
		return rate.NewLimiter(0, 0)
	}

	limiter := rate.NewLimiter(rate.Limit(l.config.Rate), l.config.Burst)
	l.visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func rateLimit(ctx context.Context, config RateLimitConfig, keyFunc func(c echo.Context) string) echo.MiddlewareFunc {
	limiter := newIPRateLimiter(ctx, config)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := keyFunc(c)
			l := limiter.getVisitor(key)
			if !l.Allow() {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			return next(c)
		}
	}
}

// PublicRateLimiter limits requests by client IP. The cleanup goroutine stops when ctx is cancelled.
func PublicRateLimiter(ctx context.Context, config RateLimitConfig) echo.MiddlewareFunc {
	return rateLimit(ctx, config, func(c echo.Context) string {
		return c.RealIP()
	})
}

// AuthRateLimiter limits requests by authenticated user ID (falls back to IP).
// The cleanup goroutine stops when ctx is cancelled.
func AuthRateLimiter(ctx context.Context, config RateLimitConfig) echo.MiddlewareFunc {
	return rateLimit(ctx, config, func(c echo.Context) string {
		if u, ok := UserFromContext(c); ok {
			return u.ID.String()
		}
		return c.RealIP()
	})
}
