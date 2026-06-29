package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/pulse/internal/db"
	"github.com/mnah05/pulse/internal/dto"
	"github.com/mnah05/pulse/internal/helper"
)

// RegistrationService provides business logic for registration operations.
type RegistrationService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	access  *EventAccessService
	log     *slog.Logger
}

// NewRegistrationService creates a RegistrationService with the given dependencies.
func NewRegistrationService(pool *pgxpool.Pool, queries *db.Queries, access *EventAccessService, log *slog.Logger) *RegistrationService {
	return &RegistrationService{pool: pool, queries: queries, access: access, log: log}
}

// ---------------------------------------------------------------------------
// Guest
// ---------------------------------------------------------------------------

// CreateGuest registers a guest for the given event. The registration
// status defaults to "approved" unless the event requires approval.
func (s *RegistrationService) CreateGuest(ctx context.Context, eventID uuid.UUID, req dto.CreateGuestRegistrationRequest) (db.Registration, error) {
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
		Name:           req.Name,
		Email:          req.Email,
		Status:         status,
		AccessToken:    helper.GenerateAccessToken(),
		TokenExpiresAt: tokenExpires,
	}

	reg, err := insertGuestRegistration(ctx, qtx, params)
	if err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
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
func (s *RegistrationService) CancelGuest(ctx context.Context, token uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.CancelGuestRegistration(ctx, token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("guest registration with token %s: %w", token, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("cancel guest registration: %w", err)
	}
	return reg, nil
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

// Create registers an authenticated user for the given event. Name and
// email fall back to the user's profile values when not provided in req.
func (s *RegistrationService) Create(ctx context.Context, user *db.User, eventID uuid.UUID, req dto.CreateRegistrationRequest) (db.Registration, error) {
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

	email := user.Email
	if req.Email != nil && *req.Email != "" {
		email = *req.Email
	}

	params := db.CreateRegistrationUserParams{
		EventID:        eventID,
		UserID:         pgtype.UUID{Valid: true, Bytes: user.ID},
		Name:           name,
		Email:          email,
		Status:         status,
		AccessToken:    helper.GenerateAccessToken(),
		TokenExpiresAt: pgtype.Timestamptz{Valid: false},
	}

	reg, err := insertUserRegistration(ctx, qtx, params)
	if err != nil {
		return db.Registration{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return reg, nil
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
	if override != nil && *override != "" {
		return *override
	}
	if user.Username != nil && *user.Username != "" {
		return *user.Username
	}
	// Fallback: email local-part
	parts := strings.Split(user.Email, "@")
	return parts[0]
}

// Cancel cancels a user's own registration.
func (s *RegistrationService) Cancel(ctx context.Context, registrationID uuid.UUID, userID uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.CancelRegistration(ctx, db.CancelRegistrationParams{
		ID:     registrationID,
		UserID: pgtype.UUID{Valid: true, Bytes: userID},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("cancel registration: %w", err)
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
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

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

// ListByEvent returns a paginated list of registrations for the given
// event, verifying that the requester is event staff. The second return
// value is the total count.
func (s *RegistrationService) ListByEvent(ctx context.Context, eventID uuid.UUID, actorID uuid.UUID, params ListParams) ([]db.Registration, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	if _, err := s.access.RequireEventStaff(ctx, eventID, actorID); err != nil {
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

// Approve sets a registration's status to "approved".
func (s *RegistrationService) Approve(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
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
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("approve registration: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return approved, nil
}

// Reject sets a registration's status to "rejected".
func (s *RegistrationService) Reject(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID) (db.Registration, error) {
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

	reg, err := s.queries.RejectRegistration(ctx, db.RejectRegistrationParams{
		ID:          registrationID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("reject registration: %w", err)
	}
	return reg, nil
}

// UpdateStatus sets a registration's status to an arbitrary value.
func (s *RegistrationService) UpdateStatus(ctx context.Context, registrationID uuid.UUID, organizerID uuid.UUID, status string) (db.Registration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Registration{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	reg, err := qtx.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
		ID:          registrationID,
		OrganizerID: organizerID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("get registration: %w", err)
	}
	if reg.Status == status {
		if err := tx.Commit(ctx); err != nil {
			return db.Registration{}, fmt.Errorf("commit tx: %w", err)
		}
		return reg, nil
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
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Registration{}, fmt.Errorf("registration %s: %w", registrationID, ErrRegistrationNotFound)
		}
		return db.Registration{}, fmt.Errorf("update registration status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Registration{}, fmt.Errorf("commit tx: %w", err)
	}
	return updated, nil
}

// CheckIn marks an attendee as checked in. If the attendee is already
// checked in, the existing registration is returned idempotently.
func (s *RegistrationService) CheckIn(ctx context.Context, registrationID uuid.UUID, actorID uuid.UUID, accessToken uuid.UUID) (db.Registration, error) {
	reg, err := s.queries.CheckInRegistration(ctx, db.CheckInRegistrationParams{
		ID:          registrationID,
		OrganizerID: actorID,
		AccessToken: accessToken,
	})
	if err == pgx.ErrNoRows {
		// Either: (a) already checked in, or (b) not found/wrong token
		// Distinguish by fetching the registration
		existing, fetchErr := s.queries.GetRegistrationByIDAdmin(ctx, db.GetRegistrationByIDAdminParams{
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
			// Already checked in — return existing (idempotent 2xx)
			return existing, nil
		}
		event, eventErr := s.queries.GetEventByID(ctx, existing.EventID)
		if eventErr == nil && event.Status != "published" {
			return db.Registration{}, fmt.Errorf("event %s: %w", existing.EventID, ErrEventNotOpen)
		}
		return db.Registration{}, fmt.Errorf("check-in failed: %w", err)
	}
	if err != nil {
		return db.Registration{}, fmt.Errorf("check in attendee: %w", err)
	}
	return reg, nil
}
