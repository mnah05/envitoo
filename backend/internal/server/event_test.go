package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/service"
)

// ---------------------------------------------------------------------------
// Test helpers (DB-backed; skip when no local postgres is available)
// ---------------------------------------------------------------------------

func openServerTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		if u := os.Getenv("DB_URL"); isLocalPostgresURL(u) {
			dsn = u
		}
	}
	if dsn == "" {
		t.Skip("set TEST_DB_URL (or local DB_URL) to run event handler tests")
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

func setupServerTestUser(t *testing.T, pool *pgxpool.Pool, queries *db.Queries) db.User {
	t.Helper()
	u, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		AuthUserID: uuid.NewString(),
		Email:      "handler-test-" + uuid.NewString()[:8] + "@example.com",
		Username:   nil,
		AvatarUrl:  nil,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", u.ID)
	})
	return u
}

// fakeAuthUser injects the local user into Echo's context, mimicking the
// LoadUser middleware so the handler can run without Clerk.
func fakeAuthUser(u db.User) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", &u)
			return next(c)
		}
	}
}

func newEventServer(t *testing.T, pool *pgxpool.Pool, queries *db.Queries, storage *service.StorageService) *Server {
	t.Helper()
	access := service.NewEventAccessService(queries)
	return &Server{
		pool:     pool,
		queries:  queries,
		validate: dto.NewValidator(),
		log:      slog.Default(),
		events:   service.NewEventService(pool, queries, access, slog.Default()),
		storage:  storage,
	}
}

func doCreateEvent(s *Server, user db.User, body string) *httptest.ResponseRecorder {
	e := echo.New()
	e.POST("/api/events", s.CreateEvent, fakeAuthUser(user))
	req := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func doUpdateEvent(s *Server, user db.User, eventID uuid.UUID, body string) *httptest.ResponseRecorder {
	e := echo.New()
	e.PATCH("/api/events/:id", s.UpdateEvent, fakeAuthUser(user))
	req := httptest.NewRequest(http.MethodPatch, "/api/events/"+eventID.String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func validEventBody(coverURL string) string {
	cover := ""
	if coverURL != "" {
		cover = `,"cover_image_url":"` + coverURL + `"`
	}
	return `{
		"title":"Handler Test Event",
		"description":"created by handler test",
		"starts_at":"2030-01-02T10:00:00Z",
		"ends_at":"2030-01-02T12:00:00Z",
		"capacity":50,
		"price_in_paise":0,
		"currency":"INR",
		"requires_approval":false,
		"is_virtual":false,
		"venue_name":"Test Hall",
		"location_address":"123 Test Street",
		"lat":12.9716,
		"lng":77.5946
		` + cover + `
	}`
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateEvent_NullifiesUnverifiedCover(t *testing.T) {
	pool, queries := openServerTestDB(t)
	user := setupServerTestUser(t, pool, queries)
	storage := newFakeStorage(t)
	s := newEventServer(t, pool, queries, storage)

	body := validEventBody("https://evil.example.com/stolen.jpg")
	rec := doCreateEvent(s, user, body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var resp dto.EventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.CoverImageURL != nil {
		t.Fatalf("cover_image_url = %v, want null after failed verification", *resp.CoverImageURL)
	}
	if resp.Warning == nil || !strings.Contains(*resp.Warning, "verified") {
		t.Fatalf("warning = %v, want verification warning", resp.Warning)
	}

	row, err := queries.GetEventByID(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("get event from db: %v", err)
	}
	if row.CoverImageUrl != nil {
		t.Fatalf("db cover_image_url = %v, want NULL after failed verification", *row.CoverImageUrl)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM events WHERE id = $1", row.ID) })
}

func TestCreateEvent_KeepsCoverWhenStorageUnconfigured(t *testing.T) {
	pool, queries := openServerTestDB(t)
	user := setupServerTestUser(t, pool, queries)
	s := newEventServer(t, pool, queries, nil) // storage nil: verification skipped

	body := validEventBody("https://img.example.com/legit.jpg")
	rec := doCreateEvent(s, user, body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var resp dto.EventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.CoverImageURL == nil || *resp.CoverImageURL != "https://img.example.com/legit.jpg" {
		t.Fatalf("cover_image_url = %v, want preserved when storage unconfigured", resp.CoverImageURL)
	}
	if resp.Warning != nil {
		t.Fatalf("warning = %v, want none", *resp.Warning)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM events WHERE id = $1", resp.ID) })
}

func TestCreateEvent_NoCoverNoWarning(t *testing.T) {
	pool, queries := openServerTestDB(t)
	user := setupServerTestUser(t, pool, queries)
	s := newEventServer(t, pool, queries, newFakeStorage(t))

	rec := doCreateEvent(s, user, validEventBody(""))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var resp dto.EventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Warning != nil {
		t.Fatalf("warning = %v, want none without cover", *resp.Warning)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM events WHERE id = $1", resp.ID) })
}

func TestUpdateEvent_NullifiesUnverifiedCover(t *testing.T) {
	pool, queries := openServerTestDB(t)
	user := setupServerTestUser(t, pool, queries)

	// Create with a storage-less server so the cover URL is stored as-is
	// (creation verification is covered by TestCreateEvent_*).
	createRec := doCreateEvent(newEventServer(t, pool, queries, nil), user,
		validEventBody("https://cdn.example.com/event-covers/original.jpg"))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body: %s", createRec.Code, createRec.Body.String())
	}
	var created dto.EventResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM events WHERE id = $1", created.ID) })

	s := newEventServer(t, pool, queries, newFakeStorage(t))
	updateRec := doUpdateEvent(s, user, created.ID, `{"cover_image_url":"https://evil.example.com/stolen.jpg"}`)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body: %s", updateRec.Code, updateRec.Body.String())
	}
	var updated dto.EventResponse
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Warning == nil || !strings.Contains(*updated.Warning, "verified") {
		t.Fatalf("warning = %v, want verification warning", updated.Warning)
	}

	row, err := queries.GetEventByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get event from db: %v", err)
	}
	if row.CoverImageUrl == nil || *row.CoverImageUrl != "https://cdn.example.com/event-covers/original.jpg" {
		t.Fatalf("db cover_image_url = %v, want original cover kept", row.CoverImageUrl)
	}
}
