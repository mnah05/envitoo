package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/email"
	"github.com/mnah05/envitoo/internal/helper"
)

// RegistrationService provides business logic for registration operations.
// It handles two registration paths: guest (via email OTP, no db.User row)
// and authenticated user (via Clerk JWT). Both paths share the same
// underlying registration table but differ in user_id nullability.
type RegistrationService struct {
	pool      *pgxpool.Pool
	queries   *db.Queries
	access    *EventAccessService
	cache     *cache.Store
	otpSecret []byte
	log       *slog.Logger
}

// NewRegistrationService creates a RegistrationService with the given dependencies.
// otpSecret is the HMAC key for guest email OTP; empty disables OTP endpoints (503).
func NewRegistrationService(pool *pgxpool.Pool, queries *db.Queries, access *EventAccessService, c *cache.Store, otpSecret string, log *slog.Logger) *RegistrationService {
	return &RegistrationService{
		pool:      pool,
		queries:   queries,
		access:    access,
		cache:     c,
		otpSecret: []byte(otpSecret),
		log:       log,
	}
}

// ---------------------------------------------------------------------------
// Guest
// ---------------------------------------------------------------------------

// CreateGuest registers a guest for the given event after email OTP verification.
// Requires a short-lived Redis marker from VerifyGuestOTP
// (guest:verified:{event_id}:{email}). Status defaults to "approved" unless
// the event requires approval. The registration is created with user_id=NULL
// — guest rows are identified by their email and access token only.
func (s *RegistrationService) CreateGuest(ctx context.Context, eventID uuid.UUID, req dto.CreateGuestRegistrationRequest) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_guest_create_start", "event_id", eventID)
	defer func() {
		if err != nil {
			logOpOutcome(log, "registration_guest_create", err, "event_id", eventID)
			return
		}
		log.Info("registration_guest_create_ok", "event_id", eventID, "registration_id", reg.ID, "status", reg.Status)
	}()

	if !s.cache.Available() {
		return db.Registration{}, ErrDependencyUnavailable
	}

	email := normalizeGuestEmail(req.Email)
	name := strings.TrimSpace(req.Name)
	verifiedKey := guestVerifiedKey(eventID, email)

	raw, err := s.cache.GetBytes(ctx, verifiedKey)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return db.Registration{}, ErrGuestEmailNotVerified
		}
		return db.Registration{}, fmt.Errorf("get guest verified: %w", mapCacheErr(err))
	}

	var verified guestVerifiedState
	if err := json.Unmarshal(raw, &verified); err != nil {
		_ = s.cache.DeleteKeys(ctx, verifiedKey)
		return db.Registration{}, ErrGuestEmailNotVerified
	}
	if normalizeGuestEmail(verified.Email) != email {
		return db.Registration{}, ErrGuestEmailNotVerified
	}
	// Prefer the name captured at OTP request/verify time.
	if n := strings.TrimSpace(verified.Name); n != "" {
		name = n
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	event, err := s.lockEventForRegistration(ctx, qtx, eventID)
	if err != nil {
		return db.Registration{}, err
	}

	status := "approved"
	if event.RequiresApproval {
		status = "pending"
	}

	tokenExpires := pgtype.Timestamptz{
		Valid: true,
		Time:  time.Now().Add(30 * 24 * time.Hour),
	}

	params := db.CreateRegistrationGuestParams{
		EventID:        eventID,
		Name:           name,
		Email:          email,
		Status:         status,
		AccessToken:    helper.GenerateAccessToken(),
		TokenExpiresAt: tokenExpires,
	}

	reg, err = insertGuestRegistration(ctx, qtx, params)
	if err != nil {
		return db.Registration{}, err
	}

	// Outbox insert is transactional — failure rolls back the entire operation.
	if err := enqueueRegistrationCreatedEmail(ctx, qtx, reg, event); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}

	// One-time use: consume verification after a successful create.
	if err := s.cache.DeleteKeys(ctx, verifiedKey); err != nil {
		s.logger(ctx).WarnContext(ctx, "delete guest verified marker", "event_id", eventID, "error", err)
	}

	return reg, nil
}

// GetByAccessToken looks up a registration by its guest access token.
func (s *RegistrationService) GetByAccessToken(ctx context.Context, token uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.GetRegistrationByAccessToken(ctx, token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration with token %s: %w", token, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration by token: %w", err)
	}
	return reg, nil
}

// CancelGuest cancels a guest registration by its access token.
// Outbox insert is transactional — failure rolls back the entire operation.
func (s *RegistrationService) CancelGuest(ctx context.Context, token uuid.UUID) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_guest_cancel_start")
	defer func() {
		attrs := []any{}
		if reg.ID != uuid.Nil {
			attrs = append(attrs, "registration_id", reg.ID, "event_id", reg.EventID)
		}
		if err != nil {
			logOpOutcome(log, "registration_guest_cancel", err, attrs...)
			return
		}
		log.Info("registration_guest_cancel_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	reg, err = qtx.CancelGuestRegistration(ctx, token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("guest registration with token %s: %w", token, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("cancel guest registration: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, reg.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event for outbox: %w", err)
	}
	if err := enqueueEmail(ctx, qtx, reg.Email, email.TicketCancelData{
		Name:       reg.Name,
		EventTitle: event.Title,
		EventID:    reg.EventID.String(),
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

// Create registers an authenticated user for the given event. Name falls
// back to the user's profile when not provided in req. Email always comes
// from the authenticated account (never from the client, preventing email
// spoofing on registrations).
func (s *RegistrationService) Create(ctx context.Context, user *db.User, eventID uuid.UUID, req dto.CreateRegistrationRequest) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_create_start", "event_id", eventID, "user_id", user.ID)
	defer func() {
		if err != nil {
			logOpOutcome(log, "registration_create", err, "event_id", eventID, "user_id", user.ID)
			return
		}
		log.Info("registration_create_ok", "event_id", eventID, "registration_id", reg.ID, "status", reg.Status, "user_id", user.ID)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	event, err := s.lockEventForRegistration(ctx, qtx, eventID)
	if err != nil {
		return db.Registration{}, err
	}

	status := "approved"
	if event.RequiresApproval {
		status = "pending"
	}

	name := resolveName(user, req.Name)
	// Always use the verified account email (never accept a client override).
	email := normalizeGuestEmail(user.Email)

	params := db.CreateRegistrationUserParams{
		EventID:        eventID,
		UserID:         pgtype.UUID{Valid: true, Bytes: user.ID},
		Name:           name,
		Email:          email,
		Status:         status,
		AccessToken:    helper.GenerateAccessToken(),
		TokenExpiresAt: pgtype.Timestamptz{Valid: false},
	}

	reg, err = insertUserRegistration(ctx, qtx, params)
	if err != nil {
		return db.Registration{}, err
	}

	// Outbox insert is transactional — failure rolls back the entire operation.
	if err := enqueueRegistrationCreatedEmail(ctx, qtx, reg, event); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
}

// enqueueRegistrationCreatedEmail queues the email for a newly created
// registration. Pending (requires_approval) uses ticket_status_change so
// pending attendees receive "your registration is pending review" rather
// than "your ticket is confirmed".
func enqueueRegistrationCreatedEmail(ctx context.Context, qtx *db.Queries, reg db.Registration, event db.Event) error {
	return enqueueEmail(ctx, qtx, reg.Email, registrationCreatedEmailData(reg, event))
}

// registrationCreatedEmailData selects the email payload for a newly
// created registration. ticket_confirm is reserved for approved creates.
func registrationCreatedEmailData(reg db.Registration, event db.Event) email.EmailData {
	if reg.Status == "pending" {
		return email.TicketStatusChangeData{
			Name:       reg.Name,
			EventTitle: event.Title,
			EventID:    reg.EventID.String(),
			NewStatus:  reg.Status,
		}
	}
	return email.TicketConfirmData{
		Name:        reg.Name,
		EventTitle:  event.Title,
		EventID:     reg.EventID.String(),
		AccessToken: reg.AccessToken.String(),
	}
}

func (s *RegistrationService) lockEventForRegistration(ctx context.Context, qtx *db.Queries, eventID uuid.UUID) (db.Event, error) {
	event, err := qtx.GetEventByIDForUpdate(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("get event: %w", err)
	}
	if event.Status != "published" {
		return db.Event{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotOpen)
	}
	count, err := qtx.CountActiveRegistrationsByEvent(ctx, eventID)
	if err != nil {
		return db.Event{}, fmt.Errorf("count registrations: %w", err)
	}
	if count >= event.Capacity {
		return db.Event{}, fmt.Errorf("event %s: %w", eventID, ErrEventFull)
	}
	return event, nil
}

func insertGuestRegistration(ctx context.Context, qtx *db.Queries, params db.CreateRegistrationGuestParams) (db.Registration, error) {
	reg, err := qtx.CreateRegistrationGuest(ctx, params)
	if err == pgx.ErrNoRows {
		params.AccessToken = helper.GenerateAccessToken()
		reg, err = qtx.CreateRegistrationGuest(ctx, params)
	}
	if err != nil {
		return db.Registration{}, mapRegistrationInsertError(err, "create guest registration")
	}
	return reg, nil
}

func insertUserRegistration(ctx context.Context, qtx *db.Queries, params db.CreateRegistrationUserParams) (db.Registration, error) {
	reg, err := qtx.CreateRegistrationUser(ctx, params)
	if err == pgx.ErrNoRows {
		params.AccessToken = helper.GenerateAccessToken()
		reg, err = qtx.CreateRegistrationUser(ctx, params)
	}
	if err != nil {
		return db.Registration{}, mapRegistrationInsertError(err, "create registration")
	}
	return reg, nil
}

func validateCapacityForApproval(count, capacity int32, fromStatus, toStatus string) error {
	if toStatus != "approved" {
		return nil
	}
	if fromStatus == "approved" {
		return nil
	}
	if fromStatus == "pending" {
		if count > capacity {
			return ErrEventFull
		}
		return nil
	}
	if fromStatus == "rejected" || fromStatus == "cancelled" {
		if count >= capacity {
			return ErrEventFull
		}
		return nil
	}
	return nil
}

func registrationFromListRow(row db.ListMyRegistrationsRow) db.Registration {
	return db.Registration{
		ID:             row.ID,
		EventID:        row.EventID,
		UserID:         row.UserID,
		Name:           row.Name,
		Email:          row.Email,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		AccessToken:    row.AccessToken,
		TokenExpiresAt: row.TokenExpiresAt,
		CheckedInAt:    row.CheckedInAt,
	}
}

func registrationFromEventListRow(row db.ListEventRegistrationsRow) db.Registration {
	return db.Registration{
		ID:             row.ID,
		EventID:        row.EventID,
		UserID:         row.UserID,
		Name:           row.Name,
		Email:          row.Email,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		AccessToken:    row.AccessToken,
		TokenExpiresAt: row.TokenExpiresAt,
		CheckedInAt:    row.CheckedInAt,
	}
}

func mapRegistrationInsertError(err error, action string) error {
	if helper.IsRegistrationDuplicate(err) {
		return fmt.Errorf("%s: %w", action, ErrConflict)
	}
	return fmt.Errorf("%s: %w", action, err)
}

func resolveName(user *db.User, override *string) string {
	if override != nil {
		if n := strings.TrimSpace(*override); n != "" {
			return n
		}
	}
	if user.Username != nil {
		if n := strings.TrimSpace(*user.Username); n != "" {
			return n
		}
	}
	// Fallback: email local-part
	parts := strings.Split(user.Email, "@")
	return parts[0]
}

// Cancel cancels a user's own registration. Scoped to user_id so one user
// cannot cancel another's registration (the SQL WHERE clause includes both
// id and user_id). Outbox insert is transactional — failure rolls back the
// entire operation.
func (s *RegistrationService) Cancel(ctx context.Context, registrationID uuid.UUID, userID uuid.UUID) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_cancel_start", "registration_id", registrationID, "user_id", userID)
	defer func() {
		attrs := []any{"registration_id", registrationID, "user_id", userID}
		if reg.EventID != uuid.Nil {
			attrs = append(attrs, "event_id", reg.EventID)
		}
		if err != nil {
			logOpOutcome(log, "registration_cancel", err, attrs...)
			return
		}
		log.Info("registration_cancel_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	reg, err = qtx.CancelRegistration(ctx, db.CancelRegistrationParams{
		ID:     registrationID,
		UserID: pgtype.UUID{Valid: true, Bytes: userID},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("cancel registration: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, reg.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event for outbox: %w", err)
	}
	if err := enqueueEmail(ctx, qtx, reg.Email, email.TicketCancelData{
		Name:       reg.Name,
		EventTitle: event.Title,
		EventID:    reg.EventID.String(),
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
}

// GetByID returns a single registration owned by the given user.
func (s *RegistrationService) GetByID(ctx context.Context, registrationID uuid.UUID, userID uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.GetRegistrationByID(ctx, db.GetRegistrationByIDParams{
		ID:     registrationID,
		UserID: pgtype.UUID{Valid: true, Bytes: userID},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration: %w", err)
	}
	return reg, nil
}

// ListByUser returns a paginated list of registrations for the given user.
// The second return value is the total count.
func (s *RegistrationService) ListByUser(ctx context.Context, userID uuid.UUID, params ListParams) ([]db.Registration, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := max(params.Offset, 0)

	rows, err := s.queries.ListMyRegistrations(ctx, db.ListMyRegistrationsParams{
		UserID: pgtype.UUID{Valid: true, Bytes: userID},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list my registrations: %w", err)
	}

	regs := make([]db.Registration, 0, len(rows))
	var total int32
	for _, row := range rows {
		regs = append(regs, registrationFromListRow(row))
		total = row.TotalCount
	}

	return regs, total, nil
}

// ---------------------------------------------------------------------------
// Admin (Organizer)
// ---------------------------------------------------------------------------

// EnsureEventStaff returns ErrEventNotFound when actor is not organizer
// or volunteer admin for the event. Callers that cache staff-only
// payloads must invoke this before serving a shared cache entry.
func (s *RegistrationService) EnsureEventStaff(ctx context.Context, eventID, actorID uuid.UUID) error {
	_, err := s.access.RequireEventStaff(ctx, eventID, actorID)
	return err
}

// ListByEvent returns a paginated list of registrations for the given
// event, verifying that the requester is event staff. The second return
// value is the total count.
func (s *RegistrationService) ListByEvent(ctx context.Context, eventID uuid.UUID, actorID uuid.UUID, params ListParams) ([]db.Registration, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	offset := max(params.Offset, 0)

	if err := s.EnsureEventStaff(ctx, eventID, actorID); err != nil {
		return nil, 0, err
	}

	rows, err := s.queries.ListEventRegistrations(ctx, db.ListEventRegistrationsParams{
		EventID:     eventID,
		OrganizerID: actorID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list event registrations: %w", err)
	}

	regs := make([]db.Registration, 0, len(rows))
	var total int32
	for _, row := range rows {
		regs = append(regs, registrationFromEventListRow(row))
		total = row.TotalCount
	}

	return regs, total, nil
}

// GetByIDAsAdmin returns a single registration, verifying that the
// requester is event staff.
func (s *RegistrationService) GetByIDAsAdmin(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration as admin: %w", err)
	}
	return reg, nil
}

// Approve sets a registration's status to "approved". Guards against
// over-capacity approval via validateCapacityForApproval. The event row
// is locked (FOR UPDATE) within the transaction so concurrent approvals
// cannot exceed capacity.
func (s *RegistrationService) Approve(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_approve_start", "registration_id", registrationID, "actor_id", actorID)
	defer func() {
		attrs := []any{"registration_id", registrationID, "actor_id", actorID}
		if reg.EventID != uuid.Nil {
			attrs = append(attrs, "event_id", reg.EventID)
		}
		if err != nil {
			logOpOutcome(log, "registration_approve", err, attrs...)
			return
		}
		log.Info("registration_approve_ok", attrs...)
	}()

	reg, err = s.queries.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration: %w", err)
	}
	if err := s.access.RequireRegistrationModeration(ctx, reg.EventID, actorID); err != nil {
		return db.Registration{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	reg, err = qtx.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration: %w", err)
	}
	if reg.Status == "approved" {
		if err := tx.Commit(ctx); err != nil {
			return db.Registration{}, fmt.Errorf("commit tx: %w", err)
		}
		return reg, nil
	}

	event, err := qtx.GetEventByIDForUpdate(ctx, reg.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event: %w", err)
	}

	count, err := qtx.CountActiveRegistrationsByEvent(ctx, reg.EventID)
	if err != nil {
		return db.Registration{}, fmt.Errorf("count registrations: %w", err)
	}
	if err := validateCapacityForApproval(count, event.Capacity, reg.Status, "approved"); err != nil {
		return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, err)
	}

	approved, err := qtx.ApproveRegistration(ctx, db.ApproveRegistrationParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrConflict)
		}
		return db.Registration{}, fmt.Errorf("approve registration: %w", err)
	}

	// Outbox insert is transactional — failure rolls back the entire operation.
	if err := enqueueEmail(ctx, qtx, approved.Email, email.TicketStatusChangeData{
		Name:       approved.Name,
		EventTitle: event.Title,
		EventID:    approved.EventID.String(),
		NewStatus:  approved.Status,
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return approved, nil
}

// Reject sets a registration's status to "rejected". Only pending
// registrations may be rejected (pre-check done outside the transaction).
// Outbox insert is transactional — failure rolls back the entire operation.
func (s *RegistrationService) Reject(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_reject_start", "registration_id", registrationID, "actor_id", actorID)
	defer func() {
		attrs := []any{"registration_id", registrationID, "actor_id", actorID}
		if reg.EventID != uuid.Nil {
			attrs = append(attrs, "event_id", reg.EventID)
		}
		if err != nil {
			logOpOutcome(log, "registration_reject", err, attrs...)
			return
		}
		log.Info("registration_reject_ok", attrs...)
	}()

	existing, err := s.queries.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration: %w", err)
	}
	if err := s.access.RequireRegistrationModeration(ctx, existing.EventID, actorID); err != nil {
		return db.Registration{}, err
	}
	if existing.Status != "pending" {
		return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationInvalidStatus)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	reg, err = qtx.RejectRegistration(ctx, db.RejectRegistrationParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrConflict)
		}
		return db.Registration{}, fmt.Errorf("reject registration: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, reg.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event for outbox: %w", err)
	}
	if err := enqueueEmail(ctx, qtx, reg.Email, email.TicketStatusChangeData{
		Name:       reg.Name,
		EventTitle: event.Title,
		EventID:    reg.EventID.String(),
		NewStatus:  reg.Status,
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
}

// allowedRegistrationStatusTransitions is the organizer status state machine.
// Cancelled is terminal via this endpoint (re-register instead of resurrecting).
// Rejected may be re-approved to match ApproveRegistration.
var allowedRegistrationStatusTransitions = map[string]map[string]struct{}{
	"pending": {
		"approved":  {},
		"rejected":  {},
		"cancelled": {},
	},
	"approved": {
		"cancelled": {},
		"rejected":  {},
	},
	"rejected": {
		"approved": {},
	},
	"cancelled": {},
}

func registrationStatusTransitionAllowed(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := allowedRegistrationStatusTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// UpdateStatus sets a registration's status when the transition is legal,
// as defined by allowedRegistrationStatusTransitions. Organizer-only:
// volunteer admins must use the approve/reject endpoints. This enforces the
// status state machine: pending→approved, pending→rejected, approved→cancelled,
// approved→rejected, rejected→approved (re-approval path).
func (s *RegistrationService) UpdateStatus(ctx context.Context, registrationID uuid.UUID, organizerID uuid.UUID, status string) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_status_update_start", "registration_id", registrationID, "organizer_id", organizerID, "status", status)
	defer func() {
		attrs := []any{"registration_id", registrationID, "organizer_id", organizerID, "status", status}
		if reg.EventID != uuid.Nil {
			attrs = append(attrs, "event_id", reg.EventID)
		}
		if err != nil {
			logOpOutcome(log, "registration_status_update", err, attrs...)
			return
		}
		log.Info("registration_status_update_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Staff-scoped fetch first (404 when no access at all), then require
	// organizer so volunteer admins cannot no-op or mutate via this path.
	reg, err = qtx.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
		ID:          registrationID,
		OrganizerID: organizerID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration: %w", err)
	}
	isOrg, err := qtx.IsEventOrganizer(ctx, db.IsEventOrganizerParams{
		ID:          reg.EventID,
		OrganizerID: organizerID,
	})
	if err != nil {
		return db.Registration{}, fmt.Errorf("check event organizer: %w", err)
	}
	if !isOrg {
		return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
	}
	if reg.Status == status {
		if err := tx.Commit(ctx); err != nil {
			return db.Registration{}, fmt.Errorf("commit tx: %w", err)
		}
		return reg, nil
	}
	if !registrationStatusTransitionAllowed(reg.Status, status) {
		return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationInvalidStatus)
	}

	if status == "approved" {
		event, err := qtx.GetEventByIDForUpdate(ctx, reg.EventID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
			}
			return db.Registration{}, fmt.Errorf("get event: %w", err)
		}
		count, err := qtx.CountActiveRegistrationsByEvent(ctx, reg.EventID)
		if err != nil {
			return db.Registration{}, fmt.Errorf("count registrations: %w", err)
		}
		if err := validateCapacityForApproval(count, event.Capacity, reg.Status, status); err != nil {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, err)
		}
	}

	updated, err := qtx.UpdateRegistrationStatus(ctx, db.UpdateRegistrationStatusParams{
		ID:          registrationID,
		OrganizerID: organizerID,
		Status:      status,
		Status_2:    reg.Status,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrConflict)
		}
		return db.Registration{}, fmt.Errorf("update registration status: %w", err)
	}

	// Outbox insert is transactional — failure rolls back the entire operation.
	event, err := qtx.GetEventByID(ctx, updated.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", updated.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event for outbox: %w", err)
	}
	if err := enqueueEmail(ctx, qtx, updated.Email, email.TicketStatusChangeData{
		Name:       updated.Name,
		EventTitle: event.Title,
		EventID:    updated.EventID.String(),
		NewStatus:  updated.Status,
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return updated, nil
}

// CheckIn marks an attendee as checked in. Uses GET FOR UPDATE to lock
// the registration row, preventing double check-in races. If the attendee
// is already checked in, the existing registration is returned idempotently
// (after verifying staff access + access token match).
// Outbox insert is transactional — failure rolls back the entire operation.
func (s *RegistrationService) CheckIn(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID, accessToken uuid.UUID) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_checkin_start", "registration_id", registrationID, "actor_id", actorID)
	defer func() {
		attrs := []any{"registration_id", registrationID, "actor_id", actorID}
		if reg.EventID != uuid.Nil {
			attrs = append(attrs, "event_id", reg.EventID)
		}
		if err != nil {
			logOpOutcome(log, "registration_checkin", err, attrs...)
			return
		}
		log.Info("registration_checkin_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	locked, err := qtx.GetRegistrationByIDForUpdate(ctx, registrationID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("lock registration: %w", err)
	}

	if locked.CheckedInAt.Valid {
		// Idempotent return still requires staff authz + matching access token.
		// Without this, any authenticated caller who knows a registration ID
		// could read PII and the guest access_token after check-in.
		existing, fetchErr := qtx.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
			ID:          registrationID,
			OrganizerID: actorID,
		})
		if fetchErr == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		if fetchErr != nil {
			return db.Registration{}, fmt.Errorf("check-in authz: %w", fetchErr)
		}
		if existing.AccessToken != accessToken {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		if err := tx.Commit(ctx); err != nil {
			return db.Registration{}, fmt.Errorf("commit tx: %w", err)
		}
		return existing, nil
	}

	reg, err = qtx.CheckInRegistration(ctx, db.CheckInRegistrationParams{
		ID:          registrationID,
		OrganizerID: actorID,
		AccessToken: accessToken,
	})
	if err == pgx.ErrNoRows {
		existing, fetchErr := qtx.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
			ID:          registrationID,
			OrganizerID: actorID,
		})
		if fetchErr == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		if fetchErr != nil {
			return db.Registration{}, fmt.Errorf("check-in idempotency check: %w", fetchErr)
		}
		if existing.AccessToken != accessToken {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		if existing.CheckedInAt.Valid {
			if err := tx.Commit(ctx); err != nil {
				return db.Registration{}, fmt.Errorf("commit tx: %w", err)
			}
			return existing, nil
		}
		if existing.Status != "approved" {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationInvalidStatus)
		}
		event, eventErr := qtx.GetEventByID(ctx, existing.EventID)
		if eventErr != nil {
			if eventErr == pgx.ErrNoRows {
				return db.Registration{}, fmt.Errorf("event %s: %w", existing.EventID, ErrEventNotFound)
			}
			return db.Registration{}, fmt.Errorf("get event for check-in: %w", eventErr)
		}
		if event.Status != "published" {
			return db.Registration{}, fmt.Errorf("event %s: %w", existing.EventID, ErrEventNotOpen)
		}
		existing, fetchErr = qtx.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
			ID:          registrationID,
			OrganizerID: actorID,
		})
		if fetchErr == nil && existing.CheckedInAt.Valid {
			if err := tx.Commit(ctx); err != nil {
				return db.Registration{}, fmt.Errorf("commit tx: %w", err)
			}
			return existing, nil
		}
		return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
	}
	if err != nil {
		return db.Registration{}, fmt.Errorf("check in attendee: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, reg.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event for check-in outbox: %w", err)
	}
	if err := enqueueEmail(ctx, qtx, reg.Email, email.CheckInData{
		Name:       reg.Name,
		EventTitle: event.Title,
		EventID:    reg.EventID.String(),
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
}

// CheckInByEventToken checks in an attendee using the event id and QR access
// token. Resolves the registration server-side so clients do not need the
// full registration list loaded. The QR code encodes the access_token; the
// scanner knows which event it's scanning for, so no registration ID is needed.
// Outbox insert is transactional — failure rolls back the entire operation.
func (s *RegistrationService) CheckInByEventToken(ctx context.Context, eventID, actorID, accessToken uuid.UUID) (reg db.Registration, err error) {
	log := s.logger(ctx)
	log.Info("registration_checkin_by_token_start", "event_id", eventID, "actor_id", actorID)
	defer func() {
		attrs := []any{"event_id", eventID, "actor_id", actorID}
		if reg.ID != uuid.Nil {
			attrs = append(attrs, "registration_id", reg.ID)
		}
		if err != nil {
			logOpOutcome(log, "registration_checkin_by_token", err, attrs...)
			return
		}
		log.Info("registration_checkin_by_token_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	reg, err = qtx.CheckInRegistrationByEventToken(ctx, db.CheckInRegistrationByEventTokenParams{
		EventID:     eventID,
		OrganizerID: actorID,
		AccessToken: accessToken,
	})
	if err == pgx.ErrNoRows {
		existing, fetchErr := qtx.GetRegistrationByEventTokenAdmin(ctx, db.GetRegistrationByEventTokenAdminParams{
			EventID:     eventID,
			AccessToken: accessToken,
			OrganizerID: actorID,
		})
		if fetchErr == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration with token %s: %w", accessToken, ErrRegistrationNotFound)
		}
		if fetchErr != nil {
			return db.Registration{}, fmt.Errorf("check-in by token lookup: %w", fetchErr)
		}
		if existing.CheckedInAt.Valid {
			if err := tx.Commit(ctx); err != nil {
				return db.Registration{}, fmt.Errorf("commit tx: %w", err)
			}
			return existing, nil
		}
		if existing.Status != "approved" {
			return db.Registration{}, fmt.Errorf("registration %s: %w", existing.ID, ErrRegistrationInvalidStatus)
		}
		event, eventErr := qtx.GetEventByID(ctx, eventID)
		if eventErr != nil {
			if eventErr == pgx.ErrNoRows {
				return db.Registration{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
			}
			return db.Registration{}, fmt.Errorf("get event for check-in: %w", eventErr)
		}
		if event.Status != "published" {
			return db.Registration{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotOpen)
		}
		existing, fetchErr = qtx.GetRegistrationByEventTokenAdmin(ctx, db.GetRegistrationByEventTokenAdminParams{
			EventID:     eventID,
			AccessToken: accessToken,
			OrganizerID: actorID,
		})
		if fetchErr == nil && existing.CheckedInAt.Valid {
			if err := tx.Commit(ctx); err != nil {
				return db.Registration{}, fmt.Errorf("commit tx: %w", err)
			}
			return existing, nil
		}
		return db.Registration{}, fmt.Errorf("registration with token %s: %w", accessToken, ErrRegistrationNotFound)
	}
	if err != nil {
		return db.Registration{}, fmt.Errorf("check in by event token: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, reg.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("event %s: %w", reg.EventID, ErrEventNotFound)
		}
		return db.Registration{}, fmt.Errorf("get event for check-in outbox: %w", err)
	}
	if err := enqueueEmail(ctx, qtx, reg.Email, email.CheckInData{
		Name:       reg.Name,
		EventTitle: event.Title,
		EventID:    reg.EventID.String(),
	}); err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
}
