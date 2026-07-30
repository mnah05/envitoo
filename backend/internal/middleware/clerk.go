package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
)

type jwksCachingBackend struct {
	inner clerk.Backend
	mu    sync.RWMutex
	raw   json.RawMessage
	exp   time.Time
}

func (b *jwksCachingBackend) Call(ctx context.Context, req *clerk.APIRequest, resource clerk.ResponseReader) error {
	if req.Method == http.MethodGet && strings.HasSuffix(req.Path, "/jwks") {
		b.mu.RLock()
		raw := b.raw
		exp := b.exp
		b.mu.RUnlock()

		if raw != nil && time.Now().Before(exp) {
			if err := json.Unmarshal(raw, resource); err == nil {
				resource.Read(&clerk.APIResponse{StatusCode: http.StatusOK})
				return nil
			}
		}
	}

	err := b.inner.Call(ctx, req, resource)
	if err != nil {
		return err
	}

	if req.Method == http.MethodGet && strings.HasSuffix(req.Path, "/jwks") {
		if data, marshalErr := json.Marshal(resource); marshalErr == nil {
			b.mu.Lock()
			b.raw = data
			b.exp = time.Now().Add(50 * time.Minute)
			b.mu.Unlock()
		}
	}
	return nil
}

var (
	jwksWarmOnce      sync.Once
	warmedJWKSBackend *jwksCachingBackend
)

func InitJWKSClient(ctx context.Context) *jwks.Client {
	jwksWarmOnce.Do(func() {
		backend := clerk.GetBackend()
		caching := &jwksCachingBackend{inner: backend}

		warmReq := clerk.NewAPIRequest(http.MethodGet, "/jwks")
		warmSet := &clerk.JSONWebKeySet{}
		if err := caching.inner.Call(ctx, warmReq, warmSet); err != nil {
			slog.Warn("jwks pre-warm failed, will fetch on first request", "error", err)
			return
		}
		if data, err := json.Marshal(warmSet); err == nil {
			caching.mu.Lock()
			caching.raw = data
			caching.exp = time.Now().Add(50 * time.Minute)
			caching.mu.Unlock()
		}
		warmedJWKSBackend = caching
	})
	if warmedJWKSBackend == nil {
		return nil
	}
	return &jwks.Client{Backend: warmedJWKSBackend}
}
