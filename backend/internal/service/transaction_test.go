package service

import (
	"context"
	"errors"
	"os"
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

func openTransactionTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		if u := os.Getenv("DB_URL"); isLocalPostgresURL(u) {
			dsn = u
		}
	}
	if dsn == "" {
		t.Skip("set TEST_DB_URL (or local DB_URL) to run transaction tests")
		return nil, nil
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

// createTxTestUser inserts a real users row (calendars.user_id has an FK)
// and returns it along with a cleanup func that deletes its calendar.
func createTxTestUser(t *testing.T, pool *pgxpool.Pool, queries *db.Queries, prefix string) (db.User, func()) {
	t.Helper()
	ctx := context.Background()

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		AuthUserID: "auth_" + prefix,
		Email:      prefix + "@example.com",
		Username:   strPtr(prefix),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx, `DELETE FROM calendars WHERE user_id = $1`, user.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, user.ID)
	}
	return user, cleanup
}

func TestTransaction_Rollback_DoesNotPersist(t *testing.T) {
	pool, queries := openTransactionTestDB(t)
	if pool == nil {
		return
	}

	prefix := "rollback-" + uuid.NewString()[:8]
	user, cleanup := createTxTestUser(t, pool, queries, prefix)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	qtx := queries.WithTx(tx)
	if _, err := qtx.CreateCalendar(ctx, db.CreateCalendarParams{
		UserID: user.ID,
		Slug:   prefix + "-cal",
		Name:   "Rollback Test Calendar",
	}); err != nil {
		t.Fatalf("create calendar in tx: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	// After rollback the calendar must not exist.
	_, err = queries.GetCalendarByUserID(ctx, user.ID)
	if err == nil {
		t.Fatal("expected calendar to not exist after rollback")
	}

	// The user row created outside the tx must still exist.
	_, err = queries.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected user to still exist after rollback: %v", err)
	}
}

func TestTransaction_Commit_Persists(t *testing.T) {
	pool, queries := openTransactionTestDB(t)
	if pool == nil {
		return
	}

	prefix := "commit-" + uuid.NewString()[:8]
	user, cleanup := createTxTestUser(t, pool, queries, prefix)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	qtx := queries.WithTx(tx)
	created, err := qtx.CreateCalendar(ctx, db.CreateCalendarParams{
		UserID: user.ID,
		Slug:   prefix + "-cal",
		Name:   "Commit Test Calendar",
	})
	if err != nil {
		t.Fatalf("create calendar in tx: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	// After commit the calendar must exist.
	got, err := queries.GetCalendarByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get calendar after commit: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("calendar IDs do not match: got %v, want %v", got.ID, created.ID)
	}
}

// TestCreateRegistration_CommitPersistsRegistrationAndOutbox verifies the
// transactional-outbox invariant on the success path: a committed create
// leaves both the registration row and its email_outbox row visible.
func TestCreateRegistration_CommitPersistsRegistrationAndOutbox(t *testing.T) {
	fx := setupConcurrentRegistrationFixture(t, 5, false, 1)
	ctx := context.Background()

	user := fx.users[0]
	reg, err := fx.svc.Create(ctx, &user, fx.event.ID, dto.CreateRegistrationRequest{})
	if err != nil {
		t.Fatalf("create registration: %v", err)
	}

	got, err := fx.queries.GetRegistrationByID(ctx, db.GetRegistrationByIDParams{
		ID:     reg.ID,
		UserID: pgtype.UUID{Valid: true, Bytes: user.ID},
	})
	if err != nil {
		t.Fatalf("get registration after commit: %v", err)
	}
	if got.Status != "approved" {
		t.Fatalf("status = %q, want approved", got.Status)
	}

	var outboxCount int64
	err = fx.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_outbox WHERE recipient = $1 AND status = 'PENDING'`,
		user.Email).Scan(&outboxCount)
	if err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("email_outbox rows = %d, want 1 (ticket email enqueued with registration)", outboxCount)
	}
}

// TestRollback_RegistrationAndOutbox_Atomic inserts a registration and its
// email_outbox row in one transaction, then rolls back. Neither row may
// survive: the outbox row must never be visible without its registration.
func TestRollback_RegistrationAndOutbox_Atomic(t *testing.T) {
	fx := setupConcurrentRegistrationFixture(t, 5, false, 1)
	ctx := context.Background()
	user := fx.users[0]

	tx, err := fx.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	qtx := fx.queries.WithTx(tx)

	_, err = qtx.CreateRegistrationUser(ctx, db.CreateRegistrationUserParams{
		EventID:     fx.event.ID,
		UserID:      pgtype.UUID{Valid: true, Bytes: user.ID},
		Name:        *user.Username,
		Email:       user.Email,
		Status:      "approved",
		AccessToken: helper.GenerateAccessToken(),
	})
	if err != nil {
		t.Fatalf("create registration in tx: %v", err)
	}
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    user.Email,
		Template:     "ticket_confirm",
		TemplateData: []byte(`{"name":"test"}`),
		Topic:        "emails.ticket-confirm",
	}); err != nil {
		t.Fatalf("create outbox row in tx: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	var regCount int64
	err = fx.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM registrations WHERE event_id = $1 AND user_id = $2`,
		fx.event.ID, user.ID).Scan(&regCount)
	if err != nil {
		t.Fatalf("count registrations: %v", err)
	}
	if regCount != 0 {
		t.Fatalf("registrations after rollback = %d, want 0", regCount)
	}

	var outboxCount int64
	err = fx.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_outbox WHERE recipient = $1`,
		user.Email).Scan(&outboxCount)
	if err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("email_outbox rows after rollback = %d, want 0", outboxCount)
	}
}

// TestUpdateStatus_ApprovedCapacityRace races UpdateStatus(approved) on
// rejected registrations against a capacity-1 event. Exactly one may win;
// the rest must fail with ErrEventFull (no oversell through UpdateStatus).
func TestUpdateStatus_ApprovedCapacityRace(t *testing.T) {
	const (
		capacity   = 1
		candidates = 5
	)
	fx := setupConcurrentRegistrationFixture(t, capacity, false, candidates)
	ctx := context.Background()

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
			_, err := fx.svc.UpdateStatus(ctx, id, fx.organizer.ID, "approved")
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
		t.Fatalf("active registration count = %d, want %d (oversell on UpdateStatus)", count, capacity)
	}
}
