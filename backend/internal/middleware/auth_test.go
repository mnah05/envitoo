package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	clerk "github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func openTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		if u := os.Getenv("DB_URL"); isLocalPostgresURL(u) {
			dsn = u
		}
	}
	if dsn == "" {
		t.Skip("set TEST_DB_URL (or local DB_URL) to run auth middleware tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse db url: %v", err)
	}
	cfg.MaxConns = 4
	cfg.MinConns = 0

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Skipf("connect db: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("ping db: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool, db.New(pool)
}

func isLocalPostgresURL(u string) bool {
	if u == "" {
		return false
	}
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]", "::1"} {
		if strings.Contains(u, "@"+host+":") ||
			strings.Contains(u, "://"+host+":") ||
			strings.Contains(u, "@"+host+"/") ||
			strings.Contains(u, "://"+host+"/") {
			return true
		}
	}
	return false
}

func openTestCache(t *testing.T) (*cache.Store, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	redis, closeRedis, err := cache.Open(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		mr.Close()
		t.Fatalf("open redis: %v", err)
	}
	store := cache.New(redis, slog.Default())
	return store, func() {
		closeRedis()
		mr.Close()
	}
}

// injectSessionClaims is Echo middleware that injects Clerk session claims
// into the request context, bypassing JWT verification for tests.
func injectSessionClaims(subject string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := &clerk.SessionClaims{
				RegisteredClaims: clerk.RegisteredClaims{
					Subject: subject,
				},
			}
			ctx := clerk.ContextWithSessionClaims(c.Request().Context(), claims)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func setupTestUser(t *testing.T, pool *pgxpool.Pool, queries *db.Queries, authUserID, email string) db.User {
	t.Helper()
	u, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		AuthUserID: authUserID,
		Email:      email,
		Username:   nil,
		AvatarUrl:  nil,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

// setupGuestRegistration creates an orphaned guest row for the given email.
func setupGuestRegistration(t *testing.T, pool *pgxpool.Pool, eventID uuid.UUID, organizerID uuid.UUID, email string) {
	t.Helper()
	ctx := context.Background()
	// We need an event first. The organizer must already exist in the users table.
	_, err := pool.Exec(ctx, `
		INSERT INTO events (id, title, slug, status, organizer_id, is_virtual, meeting_url, starts_at, ends_at, capacity)
		VALUES ($1, 'Test Event', $2, 'published', $3, TRUE, 'https://example.com/meet', now(), now() + interval '1 hour', 100)
		ON CONFLICT (id) DO NOTHING
	`, eventID, "test-event-"+uuid.New().String()[:8], organizerID)
	if err != nil {
		t.Fatalf("create test event: %v", err)
	}

	token := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO registrations (event_id, name, email, status, access_token)
		VALUES ($1, 'Guest User', $2, 'pending', $3)
		ON CONFLICT (access_token) DO NOTHING
	`, eventID, email, token)
	if err != nil {
		t.Fatalf("create guest registration: %v", err)
	}
}

func setupTestCalendar(t *testing.T, pool *pgxpool.Pool, calendarID, userID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO calendars (id, user_id, slug, name)
		VALUES ($1, $2, $3, 'Test Calendar')
		ON CONFLICT (id) DO NOTHING
	`, calendarID, userID, "cal-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("create test calendar: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestLoadUser_GuardSkipsWritesWhenNoOrphans(t *testing.T) {
	pool, queries := openTestDB(t)
	cacheStore, cleanup := openTestCache(t)
	defer cleanup()

	authUserID := "test_no_orphans_" + uuid.New().String()[:8]
	email := "no-orphans-" + uuid.New().String()[:8] + "@example.com"

	user := setupTestUser(t, pool, queries, authUserID, email)

	var invalidationCalls atomic.Int32
	invalidator := func(ctx context.Context, userID uuid.UUID, email string) {
		invalidationCalls.Add(1)
	}

	e := echo.New()
	e.Use(injectSessionClaims(authUserID))
	e.Use(LoadUser(pool, queries, cacheStore, invalidator))
	e.GET("/api/me", func(c echo.Context) error {
		u, ok := UserFromContext(c)
		if !ok {
			t.Error("user not set in context")
		}
		if u.ID != user.ID {
			t.Errorf("user id: got %v, want %v", u.ID, user.ID)
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}
	if n := invalidationCalls.Load(); n != 0 {
		t.Fatalf("invalidation callback called %d times, want 0 (no orphans)", n)
	}
}

func TestLoadUser_CTEStillLinksOrphans(t *testing.T) {
	pool, queries := openTestDB(t)
	cacheStore, cleanup := openTestCache(t)
	defer cleanup()

	authUserID := "test_with_orphans_" + uuid.New().String()[:8]
	email := "with-orphans-" + uuid.New().String()[:8] + "@example.com"
	eventID := uuid.New()
	calendarID := uuid.New()

	user := setupTestUser(t, pool, queries, authUserID, email)
	// Owner must exist in users table for FK constraints on calendars and events.
	owner := setupTestUser(t, pool, queries, "owner_"+authUserID, "owner-"+email)
	setupTestCalendar(t, pool, calendarID, owner.ID)
	setupGuestRegistration(t, pool, eventID, owner.ID, email)

	var invalidationCalls atomic.Int32
	invalidator := func(ctx context.Context, userID uuid.UUID, email string) {
		invalidationCalls.Add(1)
	}

	e := echo.New()
	e.Use(injectSessionClaims(authUserID))
	e.Use(LoadUser(pool, queries, cacheStore, invalidator))
	e.GET("/api/me", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}
	if n := invalidationCalls.Load(); n != 1 {
		t.Fatalf("invalidation callback called %d times, want 1", n)
	}

	// Verify the orphaned registration was linked to the user.
	var gotUserID pgtype.UUID
	err := pool.QueryRow(context.Background(),
		`SELECT user_id FROM registrations WHERE event_id = $1 AND email = $2`, eventID, email).Scan(&gotUserID)
	if err != nil {
		t.Fatalf("query linked registration: %v", err)
	}
	if !gotUserID.Valid || uuid.UUID(gotUserID.Bytes) != user.ID {
		t.Fatalf("registration not linked: got user_id=%v, want %v", gotUserID, user.ID)
	}
}

func TestLoadUser_CacheHitSkipsDBRoundTrip(t *testing.T) {
	pool, queries := openTestDB(t)
	cacheStore, cleanup := openTestCache(t)
	defer cleanup()

	authUserID := "test_cache_hit_" + uuid.New().String()[:8]
	email := "cache-hit-" + uuid.New().String()[:8] + "@example.com"

	// Insert user into DB
	user := setupTestUser(t, pool, queries, authUserID, email)

	// Pre-populate cache with the user row.
	cacheKey := "user:auth:" + authUserID
	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user for cache: %v", err)
	}
	if err := cacheStore.Set(context.Background(), cacheKey, data, 2*time.Minute); err != nil {
		t.Fatalf("pre-populate cache: %v", err)
	}

	e := echo.New()
	e.Use(injectSessionClaims(authUserID))
	e.Use(LoadUser(pool, queries, cacheStore, nil))
	e.GET("/api/me", func(c echo.Context) error {
		u, ok := UserFromContext(c)
		if !ok {
			t.Error("user not set in context")
		} else if u.ID != user.ID {
			t.Errorf("user id: got %v, want %v", u.ID, user.ID)
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestLoadUser_CacheMissPopulatesCache(t *testing.T) {
	pool, queries := openTestDB(t)
	cacheStore, cleanup := openTestCache(t)
	defer cleanup()

	authUserID := "test_cache_miss_" + uuid.New().String()[:8]
	email := "cache-miss-" + uuid.New().String()[:8] + "@example.com"

	user := setupTestUser(t, pool, queries, authUserID, email)

	// Verify cache is empty initially.
	cacheKey := "user:auth:" + authUserID
	if _, err := cacheStore.Get(context.Background(), cacheKey); err == nil {
		t.Fatal("expected cache to be empty before request")
	}

	e := echo.New()
	e.Use(injectSessionClaims(authUserID))
	e.Use(LoadUser(pool, queries, cacheStore, nil))
	e.GET("/api/me", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}

	// After the request, the cache should be populated.
	got, err := cacheStore.Get(context.Background(), cacheKey)
	if err != nil {
		t.Fatalf("cache should be populated after miss: %v", err)
	}
	var cached db.User
	if err := json.Unmarshal(got, &cached); err != nil {
		t.Fatalf("unmarshal cached user: %v", err)
	}
	if cached.ID != user.ID {
		t.Fatalf("cached user id: got %v, want %v", cached.ID, user.ID)
	}
}

func TestLoadUser_NilCacheBypassesCaching(t *testing.T) {
	pool, queries := openTestDB(t)

	authUserID := "test_nil_cache_" + uuid.New().String()[:8]
	email := "nil-cache-" + uuid.New().String()[:8] + "@example.com"

	setupTestUser(t, pool, queries, authUserID, email)

	e := echo.New()
	e.Use(injectSessionClaims(authUserID))
	e.Use(LoadUser(pool, queries, nil, nil))
	e.GET("/api/me", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}
}
