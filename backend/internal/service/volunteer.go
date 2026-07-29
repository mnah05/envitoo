package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/helper"
)

// VolunteerService provides business logic for volunteer workflow operations.
type VolunteerService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	access  *EventAccessService
	log     *slog.Logger
}

// NewVolunteerService creates a VolunteerService with the given dependencies.
func NewVolunteerService(pool *pgxpool.Pool, queries *db.Queries, access *EventAccessService, log *slog.Logger) *VolunteerService {
	return &VolunteerService{pool: pool, queries: queries, access: access, log: log}
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// UpsertSettings creates or updates volunteer settings for an event.
// Verifies that the requester is the event organizer.
func (s *VolunteerService) UpsertSettings(ctx context.Context, eventID, organizerID uuid.UUID, req dto.UpsertVolunteerSettingsRequest) (settings db.VolunteerSetting, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_settings_upsert_start", "event_id", eventID, "organizer_id", organizerID)
	defer func() {
		attrs := []any{"event_id", eventID, "organizer_id", organizerID}
		if err != nil {
			logOpOutcome(log, "volunteer_settings_upsert", err, attrs...)
			return
		}
		log.Info("volunteer_settings_upsert_ok", attrs...)
	}()

	event, err := s.queries.GetEventByID(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.VolunteerSetting{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
		}
		return db.VolunteerSetting{}, fmt.Errorf("get event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return db.VolunteerSetting{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
	}

	settings, err = s.queries.UpsertVolunteerSettings(ctx, db.UpsertVolunteerSettingsParams{
		EventID:         eventID,
		Enabled:         req.Enabled,
		Description:     req.Description,
		MaxApplications: req.MaxApplications,
	})
	if err != nil {
		return db.VolunteerSetting{}, fmt.Errorf("upsert volunteer settings: %w", err)
	}
	return settings, nil
}

// GetSettings returns volunteer settings for an event.
func (s *VolunteerService) GetSettings(ctx context.Context, eventID uuid.UUID) (db.VolunteerSetting, error) {
	settings, err := s.queries.GetVolunteerSettingsByEvent(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.VolunteerSetting{}, fmt.Errorf("event %s: %w", eventID, ErrVolunteerSettingsNotFound)
		}
		return db.VolunteerSetting{}, fmt.Errorf("get volunteer settings: %w", err)
	}
	return settings, nil
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

// CreateRole adds a volunteer role to an event. Verifies organizer.
func (s *VolunteerService) CreateRole(ctx context.Context, eventID, organizerID uuid.UUID, req dto.CreateVolunteerRoleRequest) (role db.VolunteerRole, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_role_create_start", "event_id", eventID, "organizer_id", organizerID)
	defer func() {
		attrs := []any{"event_id", eventID, "organizer_id", organizerID}
		if role.ID != uuid.Nil {
			attrs = append(attrs, "role_id", role.ID)
		}
		if err != nil {
			logOpOutcome(log, "volunteer_role_create", err, attrs...)
			return
		}
		log.Info("volunteer_role_create_ok", attrs...)
	}()

	event, err := s.queries.GetEventByID(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.VolunteerRole{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
		}
		return db.VolunteerRole{}, fmt.Errorf("get event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return db.VolunteerRole{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
	}

	role, err = s.queries.CreateVolunteerRole(ctx, db.CreateVolunteerRoleParams{
		EventID: eventID,
		Name:    req.Name,
		IsAdmin: req.IsAdmin,
	})
	if err != nil {
		if helper.IsUniqueViolation(err, "volunteer_roles_event_id_name_key") {
			return db.VolunteerRole{}, fmt.Errorf("create volunteer role: %w", ErrVolunteerRoleDuplicate)
		}
		return db.VolunteerRole{}, fmt.Errorf("create volunteer role: %w", err)
	}
	return role, nil
}

// DeleteRole removes a volunteer role. Verifies organizer and that the role
// belongs to the event named in the request path.
func (s *VolunteerService) DeleteRole(ctx context.Context, eventID, roleID, organizerID uuid.UUID) (deleted db.VolunteerRole, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_role_delete_start", "event_id", eventID, "role_id", roleID, "organizer_id", organizerID)
	defer func() {
		attrs := []any{"event_id", eventID, "role_id", roleID, "organizer_id", organizerID}
		if err != nil {
			logOpOutcome(log, "volunteer_role_delete", err, attrs...)
			return
		}
		log.Info("volunteer_role_delete_ok", attrs...)
	}()

	role, err := s.queries.GetVolunteerRoleByID(ctx, roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.VolunteerRole{}, fmt.Errorf("role %s: %w", roleID, ErrVolunteerRoleNotFound)
		}
		return db.VolunteerRole{}, fmt.Errorf("get role: %w", err)
	}
	if role.EventID != eventID {
		return db.VolunteerRole{}, fmt.Errorf("role %s: %w", roleID, ErrVolunteerRoleNotFound)
	}

	event, err := s.queries.GetEventByID(ctx, role.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.VolunteerRole{}, fmt.Errorf("event %s: %w", role.EventID, ErrEventNotFound)
		}
		return db.VolunteerRole{}, fmt.Errorf("get event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return db.VolunteerRole{}, fmt.Errorf("role %s: %w", roleID, ErrVolunteerRoleNotFound)
	}

	deleted, err = s.queries.DeleteVolunteerRole(ctx, db.DeleteVolunteerRoleParams{
		ID:      roleID,
		EventID: role.EventID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.VolunteerRole{}, fmt.Errorf("role %s: %w", roleID, ErrVolunteerRoleNotFound)
		}
		if helper.IsForeignKeyViolation(err) {
			return db.VolunteerRole{}, fmt.Errorf("delete volunteer role: %w", ErrVolunteerRoleInUse)
		}
		return db.VolunteerRole{}, fmt.Errorf("delete volunteer role: %w", err)
	}
	return deleted, nil
}

// ListRolesByEvent returns all volunteer roles for an event.
func (s *VolunteerService) ListRolesByEvent(ctx context.Context, eventID uuid.UUID) ([]db.VolunteerRole, error) {
	roles, err := s.queries.ListVolunteerRolesByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("list volunteer roles: %w", err)
	}
	return roles, nil
}

// ListStaffEventsByUser returns events where the user is an approved volunteer admin.
func (s *VolunteerService) ListStaffEventsByUser(ctx context.Context, userID uuid.UUID, params ListParams) ([]dto.EventStaffItemResponse, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := max(params.Offset, 0)

	rows, err := s.queries.ListMyVolunteerStaffEvents(ctx, db.ListMyVolunteerStaffEventsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list my volunteer staff events: %w", err)
	}

	items := make([]dto.EventStaffItemResponse, 0, len(rows))
	var total int32
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, dto.EventStaffItemResponse{
			EventID:       row.EventID,
			Title:         row.Title,
			Slug:          row.Slug,
			StartsAt:      row.StartsAt,
			EndsAt:        row.EndsAt,
			EventStatus:   row.EventStatus,
			ApplicationID: row.ApplicationID,
			RoleID:        row.RoleID,
			RoleName:      row.RoleName,
			IsAdmin:       row.IsAdmin,
			ApprovedAt:    row.ApprovedAt,
		})
	}

	return items, total, nil
}

// ---------------------------------------------------------------------------
// Applications — Volunteer
// ---------------------------------------------------------------------------

// Apply allows an authenticated user to apply to volunteer at an event.
func (s *VolunteerService) Apply(ctx context.Context, user *db.User, eventID uuid.UUID, req dto.CreateVolunteerApplicationRequest) (resp dto.VolunteerApplicationResponse, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_apply_start", "event_id", eventID, "user_id", user.ID)
	defer func() {
		attrs := []any{"event_id", eventID, "user_id", user.ID}
		if resp.ID != uuid.Nil {
			attrs = append(attrs, "application_id", resp.ID)
		}
		if err != nil {
			logOpOutcome(log, "volunteer_apply", err, attrs...)
			return
		}
		log.Info("volunteer_apply_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Lock event and verify it's published
	_, err = s.lockEventForVolunteer(ctx, qtx, eventID)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, err
	}

	// Check volunteer settings exist and are enabled
	settings, err := qtx.GetVolunteerSettingsByEvent(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", eventID, ErrVolunteerNotOpen)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get volunteer settings: %w", err)
	}
	if !settings.Enabled {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", eventID, ErrVolunteerNotOpen)
	}

	// Check max applications cap
	if settings.MaxApplications != nil && *settings.MaxApplications > 0 {
		count, err := qtx.CountVolunteerApplicationsByEvent(ctx, eventID)
		if err != nil {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("count applications: %w", err)
		}
		if count >= *settings.MaxApplications {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", eventID, ErrVolunteerMaxReached)
		}
	}

	// Verify role belongs to this event
	role, err := qtx.GetVolunteerRoleByID(ctx, req.RoleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("role %s: %w", req.RoleID, ErrVolunteerRoleNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get volunteer role: %w", err)
	}
	if role.EventID != eventID {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("role %s: %w", req.RoleID, ErrVolunteerRoleNotFound)
	}

	name := resolveName(user, req.Name)
	// Always use the verified account email (same policy as registration).
	email := normalizeGuestEmail(user.Email)

	app, err := qtx.CreateVolunteerApplication(ctx, db.CreateVolunteerApplicationParams{
		EventID:     eventID,
		UserID:      user.ID,
		RoleID:      req.RoleID,
		Name:        name,
		Email:       email,
		Phone:       req.Phone,
		Description: req.Description,
		Status:      "pending",
	})
	if err != nil {
		if helper.IsVolunteerApplicationDuplicate(err) {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("apply: %w", ErrVolunteerDuplicate)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("create volunteer application: %w", err)
	}

	// Outbox insert is transactional — failure rolls back the entire operation.
	eventForTemplate, err := qtx.GetEventByID(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get event for outbox: %w", err)
	}
	templateData, err := json.Marshal(applicationConfirmData{
		Name:       app.Name,
		EventTitle: eventForTemplate.Title,
		EventID:    eventID.String(),
		RoleName:   role.Name,
	})
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("marshal template data: %w", err)
	}
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    app.Email,
		Template:     EmailTemplateApplicationConfirm,
		TemplateData: templateData,
		Topic:        EmailTopicApplicationConfirm,
	}); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("create email outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	return volunteerAppResponse(app, role.Name, role.IsAdmin), nil
}

// ListByUser returns paginated volunteer applications for the given user.
func (s *VolunteerService) ListByUser(ctx context.Context, userID uuid.UUID, params ListParams) ([]dto.VolunteerApplicationResponse, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := max(params.Offset, 0)

	rows, err := s.queries.ListMyVolunteerApplications(ctx, db.ListMyVolunteerApplicationsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list my applications: %w", err)
	}

	items := make([]dto.VolunteerApplicationResponse, 0, len(rows))
	var total int32
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, dto.VolunteerApplicationResponse{
			ID:          row.ID,
			EventID:     row.EventID,
			UserID:      row.UserID,
			RoleID:      row.RoleID,
			RoleName:    row.RoleName,
			IsAdmin:     row.IsAdmin,
			Name:        row.Name,
			Email:       row.Email,
			Phone:       row.Phone,
			Description: row.Description,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	return items, total, nil
}

// GetByID returns a single application owned by the given user.
func (s *VolunteerService) GetByID(ctx context.Context, appID, userID uuid.UUID) (dto.VolunteerApplicationResponse, error) {
	row, err := s.queries.GetVolunteerApplicationByID(ctx, db.GetVolunteerApplicationByIDParams{
		ID:     appID,
		UserID: userID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get application: %w", err)
	}

	return dto.VolunteerApplicationResponse{
		ID:          row.ID,
		EventID:     row.EventID,
		UserID:      row.UserID,
		RoleID:      row.RoleID,
		RoleName:    row.RoleName,
		IsAdmin:     row.IsAdmin,
		Name:        row.Name,
		Email:       row.Email,
		Phone:       row.Phone,
		Description: row.Description,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// Cancel cancels a user's own pending volunteer application.
// Outbox insert is transactional — failure rolls back the entire operation.
func (s *VolunteerService) Cancel(ctx context.Context, appID, userID uuid.UUID) (resp dto.VolunteerApplicationResponse, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_cancel_start", "application_id", appID, "user_id", userID)
	defer func() {
		attrs := []any{"application_id", appID, "user_id", userID}
		if err != nil {
			logOpOutcome(log, "volunteer_cancel", err, attrs...)
			return
		}
		log.Info("volunteer_cancel_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	app, err := qtx.CancelVolunteerApplication(ctx, db.CancelVolunteerApplicationParams{
		ID:     appID,
		UserID: userID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, volunteerCancelError(ctx, qtx, appID, userID)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("cancel application: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, app.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", app.EventID, ErrEventNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get event for outbox: %w", err)
	}
	role, err := qtx.GetVolunteerRoleByID(ctx, app.RoleID)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get role for outbox: %w", err)
	}
	templateData, err := json.Marshal(applicationCancelData{
		Name:       app.Name,
		EventTitle: event.Title,
		EventID:    app.EventID.String(),
	})
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("marshal template data: %w", err)
	}
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    app.Email,
		Template:     EmailTemplateApplicationCancel,
		TemplateData: templateData,
		Topic:        EmailTopicApplicationCancel,
	}); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("create email outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	return volunteerAppResponse(app, role.Name, role.IsAdmin), nil
}

// ---------------------------------------------------------------------------
// Applications — Organizer
// ---------------------------------------------------------------------------

// ListByEvent returns paginated applications for an event. Verifies event staff.
func (s *VolunteerService) ListByEvent(ctx context.Context, eventID, actorID uuid.UUID, params ListParams) ([]dto.VolunteerApplicationResponse, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := max(params.Offset, 0)

	if _, err := s.access.RequireEventStaff(ctx, eventID, actorID); err != nil {
		return nil, 0, err
	}

	rows, err := s.queries.ListEventVolunteerApplications(ctx, db.ListEventVolunteerApplicationsParams{
		EventID:     eventID,
		OrganizerID: actorID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list event applications: %w", err)
	}

	items := make([]dto.VolunteerApplicationResponse, 0, len(rows))
	var total int32
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, dto.VolunteerApplicationResponse{
			ID:          row.ID,
			EventID:     row.EventID,
			UserID:      row.UserID,
			RoleID:      row.RoleID,
			RoleName:    row.RoleName,
			IsAdmin:     row.IsAdmin,
			Name:        row.Name,
			Email:       row.Email,
			Phone:       row.Phone,
			Description: row.Description,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	return items, total, nil
}

// ListVolunteersByEvent returns the confirmed volunteer roster for an
// event. Verifies event staff (organizer or volunteer admin).
func (s *VolunteerService) ListVolunteersByEvent(ctx context.Context, eventID, actorID uuid.UUID, params ListParams) ([]dto.VolunteerResponse, int32, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := max(params.Offset, 0)

	if _, err := s.access.RequireEventStaff(ctx, eventID, actorID); err != nil {
		return nil, 0, err
	}

	rows, err := s.queries.ListEventVolunteers(ctx, db.ListEventVolunteersParams{
		EventID:     eventID,
		OrganizerID: actorID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list event volunteers: %w", err)
	}

	items := make([]dto.VolunteerResponse, 0, len(rows))
	var total int32
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, dto.VolunteerResponse{
			ID:        row.ID,
			EventID:   row.EventID,
			UserID:    row.UserID,
			Role:      row.Role,
			Email:     row.Email,
			Username:  row.Username,
			AvatarURL: row.AvatarUrl,
			CreatedAt: row.CreatedAt,
		})
	}

	return items, total, nil
}

// GetByIDAsAdmin returns any application, verifying event staff.
func (s *VolunteerService) GetByIDAsAdmin(ctx context.Context, appID, actorID uuid.UUID) (dto.VolunteerApplicationResponse, error) {
	row, err := s.queries.GetVolunteerApplicationByIDAdmin(ctx, db.GetVolunteerApplicationByIDAdminParams{
		ID:          appID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get application as admin: %w", err)
	}

	return dto.VolunteerApplicationResponse{
		ID:          row.ID,
		EventID:     row.EventID,
		UserID:      row.UserID,
		RoleID:      row.RoleID,
		RoleName:    row.RoleName,
		IsAdmin:     row.IsAdmin,
		Name:        row.Name,
		Email:       row.Email,
		Phone:       row.Phone,
		Description: row.Description,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// Approve sets a pending application status to "approved" and records
// the confirmed volunteer in the volunteers roster table. Both operations
// run in a single transaction — if the roster insert fails, the approval
// is rolled back.
func (s *VolunteerService) Approve(ctx context.Context, appID, actorID uuid.UUID) (resp dto.VolunteerApplicationResponse, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_approve_start", "application_id", appID, "actor_id", actorID)
	defer func() {
		attrs := []any{"application_id", appID, "actor_id", actorID}
		if err != nil {
			logOpOutcome(log, "volunteer_approve", err, attrs...)
			return
		}
		log.Info("volunteer_approve_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	app, err := qtx.ApproveVolunteerApplication(ctx, db.ApproveVolunteerApplicationParams{
		ID:          appID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, volunteerModerationError(ctx, qtx, appID, actorID)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("approve application: %w", err)
	}

	role, err := qtx.GetVolunteerRoleByID(ctx, app.RoleID)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get volunteer role for roster: %w", err)
	}

	_, err = qtx.InsertVolunteer(ctx, db.InsertVolunteerParams{
		EventID: app.EventID,
		UserID:  app.UserID,
		Role:    role.Name,
	})
	if err != nil {
		if err != pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("insert volunteer roster: %w", err)
		}
	}

	// Outbox insert is transactional — failure rolls back the entire operation.
	eventForTemplate, err := qtx.GetEventByID(ctx, app.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", app.EventID, ErrEventNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get event for outbox: %w", err)
	}
	templateData, err := json.Marshal(volunteerStatusChangeData{
		Name:       app.Name,
		EventTitle: eventForTemplate.Title,
		EventID:    app.EventID.String(),
		NewStatus:  app.Status,
		RoleName:   role.Name,
	})
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("marshal template data: %w", err)
	}
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    app.Email,
		Template:     EmailTemplateVolunteerStatusChange,
		TemplateData: templateData,
		Topic:        EmailTopicVolunteerStatusChange,
	}); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("create email outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	// Build response from data already loaded — never fail the success path
	// with a post-commit re-fetch error after the state change committed.
	return volunteerAppResponse(app, role.Name, role.IsAdmin), nil
}

// Reject sets a pending application status to "rejected".
// Outbox insert is transactional — failure rolls back the entire operation.
func (s *VolunteerService) Reject(ctx context.Context, appID, actorID uuid.UUID) (resp dto.VolunteerApplicationResponse, err error) {
	log := s.logger(ctx)
	log.Info("volunteer_reject_start", "application_id", appID, "actor_id", actorID)
	defer func() {
		attrs := []any{"application_id", appID, "actor_id", actorID}
		if err != nil {
			logOpOutcome(log, "volunteer_reject", err, attrs...)
			return
		}
		log.Info("volunteer_reject_ok", attrs...)
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	app, err := qtx.RejectVolunteerApplication(ctx, db.RejectVolunteerApplicationParams{
		ID:          appID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, volunteerModerationError(ctx, qtx, appID, actorID)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("reject application: %w", err)
	}

	event, err := qtx.GetEventByID(ctx, app.EventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("event %s: %w", app.EventID, ErrEventNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get event for outbox: %w", err)
	}
	role, err := qtx.GetVolunteerRoleByID(ctx, app.RoleID)
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get role for outbox: %w", err)
	}
	templateData, err := json.Marshal(volunteerStatusChangeData{
		Name:       app.Name,
		EventTitle: event.Title,
		EventID:    app.EventID.String(),
		NewStatus:  app.Status,
		RoleName:   role.Name,
	})
	if err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("marshal template data: %w", err)
	}
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    app.Email,
		Template:     EmailTemplateVolunteerStatusChange,
		TemplateData: templateData,
		Topic:        EmailTopicVolunteerStatusChange,
	}); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("create email outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("commit tx: %w", err)
	}

	return volunteerAppResponse(app, role.Name, role.IsAdmin), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func volunteerAppResponse(app db.VolunteerApplication, roleName string, isAdmin bool) dto.VolunteerApplicationResponse {
	return dto.VolunteerApplicationResponse{
		ID:          app.ID,
		EventID:     app.EventID,
		UserID:      app.UserID,
		RoleID:      app.RoleID,
		RoleName:    roleName,
		IsAdmin:     isAdmin,
		Name:        app.Name,
		Email:       app.Email,
		Phone:       app.Phone,
		Description: app.Description,
		Status:      app.Status,
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
	}
}

// responseForApplication attaches role fields after a successful mutation.
// Role lookup failure is degraded (empty role metadata) so clients never
// see 500 after the application row already changed.
func (s *VolunteerService) responseForApplication(ctx context.Context, app db.VolunteerApplication) dto.VolunteerApplicationResponse {
	roleName, isAdmin := "", false
	if role, err := s.queries.GetVolunteerRoleByID(ctx, app.RoleID); err == nil {
		roleName, isAdmin = role.Name, role.IsAdmin
	} else {
		s.logger(ctx).WarnContext(ctx, "load role for application response", "app_id", app.ID, "role_id", app.RoleID, "error", err)
	}
	return volunteerAppResponse(app, roleName, isAdmin)
}

func (s *VolunteerService) getApplicationWithRole(ctx context.Context, appID uuid.UUID) (dto.VolunteerApplicationResponse, error) {
	row, err := s.queries.GetVolunteerApplicationWithRole(ctx, appID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return dto.VolunteerApplicationResponse{}, fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
		}
		return dto.VolunteerApplicationResponse{}, fmt.Errorf("get application with role: %w", err)
	}

	return dto.VolunteerApplicationResponse{
		ID:          row.ID,
		EventID:     row.EventID,
		UserID:      row.UserID,
		RoleID:      row.RoleID,
		RoleName:    row.RoleName,
		IsAdmin:     row.IsAdmin,
		Name:        row.Name,
		Email:       row.Email,
		Phone:       row.Phone,
		Description: row.Description,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func volunteerCancelError(ctx context.Context, queries *db.Queries, appID, userID uuid.UUID) error {
	existing, err := queries.GetVolunteerApplicationByID(ctx, db.GetVolunteerApplicationByIDParams{
		ID:     appID,
		UserID: userID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
		}
		return fmt.Errorf("get application: %w", err)
	}
	if existing.Status != "pending" {
		return fmt.Errorf("application %s: %w", appID, ErrVolunteerInvalidStatus)
	}
	return fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
}

func volunteerModerationError(ctx context.Context, queries *db.Queries, appID, actorID uuid.UUID) error {
	existing, err := queries.GetVolunteerApplicationByIDAdmin(ctx, db.GetVolunteerApplicationByIDAdminParams{
		ID:          appID,
		OrganizerID: actorID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
		}
		return fmt.Errorf("get application as admin: %w", err)
	}
	if existing.Status != "pending" {
		return fmt.Errorf("application %s: %w", appID, ErrVolunteerInvalidStatus)
	}
	// App exists and is pending, but the restricted UPDATE matched nothing.
	// Distinguish "lost access" (404) from "admin-role moderation by
	// non-organizer" (403).
	isOrg, err := queries.IsEventOrganizer(ctx, db.IsEventOrganizerParams{
		ID:          existing.EventID,
		OrganizerID: actorID,
	})
	if err != nil {
		return fmt.Errorf("check event organizer: %w", err)
	}
	if !isOrg && existing.IsAdmin {
		return fmt.Errorf("application %s: %w", appID, ErrVolunteerAdminRoleRestricted)
	}
	return fmt.Errorf("application %s: %w", appID, ErrVolunteerAppNotFound)
}

func (s *VolunteerService) lockEventForVolunteer(ctx context.Context, qtx *db.Queries, eventID uuid.UUID) (db.Event, error) {
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
	return event, nil
}

// ---------------------------------------------------------------------------
// GetVolunteerRoleByID query (needed for DeleteRole validation)
// ---------------------------------------------------------------------------
