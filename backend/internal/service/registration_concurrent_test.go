package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/helper"
)

// openConcurrentTestDB connects when TEST_DB_URL is set (preferred), or when
// DB_URL points at localhost. Remote/shared databases are never used by default
// so `go test ./...` stays safe without an explicit opt-in.
func openConcurrentTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		if u := os.Getenv("DB_URL"); isLocalPostgresURL(u) {
			dsn = u
		}
	}
	if dsn == "" {
		t.Skip("set TEST_DB_URL (or local DB_URL) to run concurrent registration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse db url: %v", err)
	}
	// Keep the pool small enough for hosted session poolers (e.g. Supabase
	// free tier ~15) while still allowing overlapping transactions.
	cfg.MaxConns = 8
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

type concurrentFixture struct {
	pool      *pgxpool.Pool
	queries   *db.Queries
	svc       *RegistrationService
	organizer db.User
	event     db.Event
	users     []db.User
	prefix    string
}

func setupConcurrentRegistrationFixture(t *testing.T, capacity int32, requiresApproval bool, registrantCount int) *concurrentFixture {
	t.Helper()

	pool, queries := openConcurrentTestDB(t)
	ctx := context.Background()
	prefix := fmt.Sprintf("conc-%s", uuid.NewString()[:8])

	organizer, err := queries.CreateUser(ctx, db.CreateUserParams{
		AuthUserID: "auth_" + prefix + "_org",
		Email:      prefix + "-org@example.com",
		Username:   strPtr(prefix + "-org"),
	})
	if err != nil {
		t.Fatalf("create organizer: %v", err)
	}

	cal, err := queries.CreateCalendar(ctx, db.CreateCalendarParams{
		UserID: organizer.ID,
		Slug:   prefix + "-cal",
		Name:   prefix + " calendar",
	})
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}

	now := time.Now().UTC()
	event, err := queries.CreateEvent(ctx, db.CreateEventParams{
		OrganizerID:      organizer.ID,
		Title:            prefix + " concurrent event",
		Description:      strPtr("capacity race fixture"),
		StartsAt:         now.Add(24 * time.Hour),
		EndsAt:           now.Add(26 * time.Hour),
		Capacity:         capacity,
		PriceInPaise:     0,
		Currency:         "INR",
		Status:           "published",
		RequiresApproval: requiresApproval,
		CalendarID:       pgtype.UUID{Valid: true, Bytes: cal.ID},
		IsVirtual:        true,
		MeetingUrl:       strPtr("https://example.com/meet/" + prefix),
		Slug:             prefix + "-event",
		Category:         strPtr("technical"),
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	users := make([]db.User, 0, registrantCount)
	for i := 0; i < registrantCount; i++ {
		u, err := queries.CreateUser(ctx, db.CreateUserParams{
			AuthUserID: fmt.Sprintf("auth_%s_u%d", prefix, i),
			Email:      fmt.Sprintf("%s-u%d@example.com", prefix, i),
			Username:   strPtr(fmt.Sprintf("%s-u%d", prefix, i)),
		})
		if err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		users = append(users, u)
	}

	access := NewEventAccessService(queries)
	svc := NewRegistrationService(pool, queries, access, nil, "", slog.Default())

	fx := &concurrentFixture{
		pool:      pool,
		queries:   queries,
		svc:       svc,
		organizer: organizer,
		event:     event,
		users:     users,
		prefix:    prefix,
	}
	t.Cleanup(fx.cleanup)
	return fx
}

func (fx *concurrentFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Outbox rows are not FK-linked; scrub by recipient prefix.
	_, _ = fx.pool.Exec(ctx, `DELETE FROM email_outbox WHERE recipient LIKE $1`, fx.prefix+"%")
	_, _ = fx.pool.Exec(ctx, `DELETE FROM registrations WHERE event_id = $1`, fx.event.ID)
	_, _ = fx.pool.Exec(ctx, `DELETE FROM events WHERE id = $1`, fx.event.ID)
	_, _ = fx.pool.Exec(ctx, `DELETE FROM calendars WHERE user_id = $1`, fx.organizer.ID)

	ids := make([]uuid.UUID, 0, len(fx.users)+1)
	ids = append(ids, fx.organizer.ID)
	for _, u := range fx.users {
		ids = append(ids, u.ID)
	}
	_, _ = fx.pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, ids)
}

func TestConcurrentRegistration_RespectsCapacity(t *testing.T) {
	const (
		capacity   = 3
		applicants = 12
	)
	fx := setupConcurrentRegistrationFixture(t, capacity, false, applicants)
	ctx := context.Background()

	var (
		okCount   atomic.Int32
		fullCount atomic.Int32
		otherErrs atomic.Int32
		wg        sync.WaitGroup
	)

	wg.Add(applicants)
	start := make(chan struct{})
	for i := 0; i < applicants; i++ {
		user := fx.users[i]
		go func() {
			defer wg.Done()
			<-start
			_, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{})
			switch {
			case err == nil:
				okCount.Add(1)
			case errors.Is(err, ErrEventFull):
				fullCount.Add(1)
			default:
				otherErrs.Add(1)
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := otherErrs.Load(); got != 0 {
		t.Fatalf("got %d unexpected errors", got)
	}
	if got := okCount.Load(); got != capacity {
		t.Fatalf("successful registrations = %d, want %d", got, capacity)
	}
	if got := fullCount.Load(); got != applicants-capacity {
		t.Fatalf("ErrEventFull count = %d, want %d", got, applicants-capacity)
	}

	count, err := fx.queries.CountActiveRegistrationsByEvent(ctx, fx.event.ID)
	if err != nil {
		t.Fatalf("count active: %v", err)
	}
	if count != capacity {
		t.Fatalf("active registration count = %d, want %d (oversell)", count, capacity)
	}
}

func TestConcurrentRegistration_ExactCapacityAllSucceed(t *testing.T) {
	const capacity = 6
	fx := setupConcurrentRegistrationFixture(t, capacity, false, capacity)
	ctx := context.Background()

	var (
		okCount   atomic.Int32
		otherErrs atomic.Int32
		wg        sync.WaitGroup
	)

	wg.Add(capacity)
	start := make(chan struct{})
	for i := 0; i < capacity; i++ {
		user := fx.users[i]
		go func() {
			defer wg.Done()
			<-start
			_, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{})
			if err != nil {
				otherErrs.Add(1)
				t.Errorf("unexpected error: %v", err)
				return
			}
			okCount.Add(1)
		}()
	}
	close(start)
	wg.Wait()

	if otherErrs.Load() != 0 {
		t.Fatalf("got unexpected errors")
	}
	if got := okCount.Load(); got != capacity {
		t.Fatalf("successful registrations = %d, want %d", got, capacity)
	}

	count, err := fx.queries.CountActiveRegistrationsByEvent(ctx, fx.event.ID)
	if err != nil {
		t.Fatalf("count active: %v", err)
	}
	if count != capacity {
		t.Fatalf("active registration count = %d, want %d", count, capacity)
	}
}

func TestConcurrentRegistration_PendingAlsoCountsTowardCapacity(t *testing.T) {
	const (
		capacity   = 3
		applicants = 9
	)
	fx := setupConcurrentRegistrationFixture(t, capacity, true, applicants)
	ctx := context.Background()

	var okCount atomic.Int32
	var fullCount atomic.Int32
	var wg sync.WaitGroup

	wg.Add(applicants)
	start := make(chan struct{})
	for i := 0; i < applicants; i++ {
		user := fx.users[i]
		go func() {
			defer wg.Done()
			<-start
			reg, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{})
			switch {
			case err == nil:
				if reg.Status != "pending" {
					t.Errorf("status = %q, want pending", reg.Status)
				}
				okCount.Add(1)
			case errors.Is(err, ErrEventFull):
				fullCount.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := okCount.Load(); got != capacity {
		t.Fatalf("successful pending registrations = %d, want %d", got, capacity)
	}
	if got := fullCount.Load(); got != applicants-capacity {
		t.Fatalf("ErrEventFull count = %d, want %d", got, applicants-capacity)
	}

	count, err := fx.queries.CountActiveRegistrationsByEvent(ctx, fx.event.ID)
	if err != nil {
		t.Fatalf("count active: %v", err)
	}
	if count != capacity {
		t.Fatalf("active registration count = %d, want %d", count, capacity)
	}
}

func TestConcurrentApprove_RejectedDoesNotOversell(t *testing.T) {
	const (
		capacity   = 1
		candidates = 5
	)
	fx := setupConcurrentRegistrationFixture(t, capacity, false, candidates)
	ctx := context.Background()

	// Seed rejected registrations directly so capacity is free, then race Approves.
	regIDs := make([]uuid.UUID, 0, candidates)
	for i, user := range fx.users {
		var id uuid.UUID
		err := fx.pool.QueryRow(ctx, `
			INSERT INTO registrations
			  (event_id, user_id, name, email, status, access_token)
			VALUES ($1, $2, $3, $4, 'rejected', $5)
			RETURNING id
		`, fx.event.ID, user.ID, *user.Username, user.Email, helper.GenerateAccessToken()).Scan(&id)
		if err != nil {
			t.Fatalf("seed rejected registration %d: %v", i, err)
		}
		regIDs = append(regIDs, id)
	}

	var (
		okCount   atomic.Int32
		fullCount atomic.Int32
		otherErrs atomic.Int32
		wg        sync.WaitGroup
	)

	wg.Add(candidates)
	start := make(chan struct{})
	for _, regID := range regIDs {
		id := regID
		go func() {
			defer wg.Done()
			<-start
			_, err := fx.svc.Approve(ctx, id, fx.organizer.ID)
			switch {
			case err == nil:
				okCount.Add(1)
			case errors.Is(err, ErrEventFull):
				fullCount.Add(1)
			default:
				otherErrs.Add(1)
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := otherErrs.Load(); got != 0 {
		t.Fatalf("got %d unexpected errors", got)
	}
	if got := okCount.Load(); got != capacity {
		t.Fatalf("successful approvals = %d, want %d", got, capacity)
	}
	if got := fullCount.Load(); got != candidates-capacity {
		t.Fatalf("ErrEventFull count = %d, want %d", got, candidates-capacity)
	}

	count, err := fx.queries.CountActiveRegistrationsByEvent(ctx, fx.event.ID)
	if err != nil {
		t.Fatalf("count active: %v", err)
	}
	if count != capacity {
		t.Fatalf("active registration count = %d, want %d (oversell on approve)", count, capacity)
	}
}

func TestCancelThenReRegister_AllowedAfterPartialUnique(t *testing.T) {
	fx := setupConcurrentRegistrationFixture(t, 5, false, 1)
	ctx := context.Background()

	var hasActivePartial bool
	err := fx.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = current_schema()
			  AND indexname = 'idx_registrations_event_email_active'
		)
	`).Scan(&hasActivePartial)
	if err != nil {
		t.Fatalf("check migration index: %v", err)
	}
	if !hasActivePartial {
		t.Skip("apply migration 20260729120000_registration_active_partial_unique to run cancel+re-register test")
	}

	user := fx.users[0]
	first, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := fx.svc.Cancel(ctx, first.ID, user.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	second, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{})
	if err != nil {
		t.Fatalf("re-register after cancel: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("expected a new registration row after cancel, got same id")
	}
	if second.Status != "approved" {
		t.Fatalf("status = %q, want approved", second.Status)
	}

	// Active duplicate must still conflict.
	if _, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second active create error = %v, want ErrConflict", err)
	}
}
