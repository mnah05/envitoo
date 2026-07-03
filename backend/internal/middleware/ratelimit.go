package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type RateLimitConfig struct {
	Rate            float64
	Burst           int
	CleanupInterval time.Duration
	CleanupIdleTime time.Duration
}

var (
	DefaultPublicConfig = RateLimitConfig{
		Rate:            10,
		Burst:           20,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
	}
	DefaultAuthConfig = RateLimitConfig{
		Rate:            30,
		Burst:           60,
		CleanupInterval: time.Minute,
		CleanupIdleTime: 3 * time.Minute,
	}
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	config   RateLimitConfig
}

func newIPRateLimiter(config RateLimitConfig) *ipRateLimiter {
	l := &ipRateLimiter{
		visitors: make(map[string]*visitor),
		config:   config,
	}
	go l.cleanupLoop()
	return l
}

func (l *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
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

func (l *ipRateLimiter) getVisitor(key string) *rate.Limiter {
	l.mu.RLock()
	v, exists := l.visitors[key]
	if exists {
		v.lastSeen = time.Now()
		l.mu.RUnlock()
		return v.limiter
	}
	l.mu.RUnlock()

	l.mu.Lock()
	if v, exists = l.visitors[key]; exists {
		v.lastSeen = time.Now()
		l.mu.Unlock()
		return v.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(l.config.Rate), l.config.Burst)
	l.visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
	l.mu.Unlock()
	return limiter
}

func rateLimit(config RateLimitConfig, keyFunc func(c echo.Context) string) echo.MiddlewareFunc {
	limiter := newIPRateLimiter(config)
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

func PublicRateLimiter(config RateLimitConfig) echo.MiddlewareFunc {
	return rateLimit(config, func(c echo.Context) string {
		return c.RealIP()
	})
}

func AuthRateLimiter(config RateLimitConfig) echo.MiddlewareFunc {
	return rateLimit(config, func(c echo.Context) string {
		if u, ok := UserFromContext(c); ok {
			return u.ID.String()
		}
		return c.RealIP()
	})
}
