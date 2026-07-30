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
//
// Note 1: An in-process rate limiter is used instead of a Redis-based one
// (like sliding window counters) to keep things simple and avoid a Redis
// dependency for basic abuse protection. The trade-off is that limiting
// resets on process restart.
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
	// DefaultPublicConfig: ~10 req/s with up to 20 burst. Suitable for
	// general-purpose API endpoints accessed by anonymous users.
	DefaultPublicConfig = RateLimitConfig{
		Rate:            10,
		Burst:           20,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
		MaxVisitors:     defaultMaxVisitors,
	}
	// DefaultOTPConfig is intentionally tighter than DefaultPublicConfig —
	// each OTP request can enqueue a real outbound email.
	//
	// Note 2: The rate is 5/60 so it's expressed as 5 requests per minute
	// (rate.Limit is events per second). The burst of 5 matches the rate
	// window so a burst does not exceed the minute quota.
	DefaultOTPConfig = RateLimitConfig{
		Rate:            5.0 / 60.0,
		Burst:           5,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
		MaxVisitors:     defaultMaxVisitors,
	}
	// DefaultAuthConfig: 30 req/s for authenticated users. Higher than
	// public because authenticated endpoints serve first-party clients
	// and can be trusted more than anonymous traffic.
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

// ipRateLimiter manages per-key rate limiters, keyed by IP or user ID.
//
// Note 3: Go's x/time/rate package implements a token bucket algorithm.
// Each key starts with Burst tokens and refills at Rate tokens per second.
// Allow() consumes one token if any remain; the bucket never exceeds Burst.
//
// Note 4: The cleanup goroutine prevents unbounded memory growth. Without
// it, a million unique IPs would create a million rate.Limiter objects that
// are never freed.
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
	//
	// Note 5: A zero-value rate.Limiter (limit 0, burst 0) never allows any
	// event. This is the hard cap behavior — once the map is full, new keys
	// are rejected until idle entries are cleaned up by the background loop.
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
//
// Note 6: Using RealIP (which respects X-Forwarded-For behind a reverse
// proxy) rather than RemoteAddr is important because all production traffic
// arrives through a load balancer or reverse proxy.
func PublicRateLimiter(ctx context.Context, config RateLimitConfig) echo.MiddlewareFunc {
	return rateLimit(ctx, config, func(c echo.Context) string {
		return c.RealIP()
	})
}

// AuthRateLimiter limits requests by authenticated user ID (falls back to IP).
// The cleanup goroutine stops when ctx is cancelled.
//
// Note 7: Keying by user ID rather than IP means a user on a shared NAT gets
// their own rate limit bucket. The fallback to IP is for authenticated
// requests where LoadUser has not run yet (e.g. during first-login
// onboarding within the same request).
func AuthRateLimiter(ctx context.Context, config RateLimitConfig) echo.MiddlewareFunc {
	return rateLimit(ctx, config, func(c echo.Context) string {
		if u, ok := UserFromContext(c); ok {
			return u.ID.String()
		}
		return c.RealIP()
	})
}
