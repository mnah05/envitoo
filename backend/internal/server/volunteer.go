package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// ---------------------------------------------------------------------------
// Public Routes
// ---------------------------------------------------------------------------

// GetVolunteerSettings handles GET /api/events/:id/volunteer/settings.
// Returns whether volunteer applications are open for the event and
// the description shown to prospective volunteers. Public endpoint.
func (s *Server) GetVolunteerSettings(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	ctx := c.Request().Context()
	resp, err := cache.GetJSON(ctx, s.cache, "GET:/api/events/"+eventID.String()+"/volunteer/settings", cache.VolunteerTTL, func(ctx context.Context) (dto.VolunteerSettingsResponse, error) {
		settings, err := s.volunteers.GetSettings(ctx, eventID)
		if err != nil {
			return dto.VolunteerSettingsResponse{}, err
		}
		return dto.NewVolunteerSettingsResponse(settings), nil
	})
	if err != nil {
		if errors.Is(err, service.ErrVolunteerSettingsNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "volunteer settings not found")
		}
		log.ErrorContext(ctx, "get volunteer settings", "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ListVolunteerRoles handles GET /api/events/:id/volunteer/roles.
// Returns the role catalog for the event (e.g. "Usher", "Stage Crew").
// Each role can allow multiple volunteers. Public endpoint.
func (s *Server) ListVolunteerRoles(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	ctx := c.Request().Context()
	resp, err := cache.GetJSON(ctx, s.cache, "GET:/api/events/"+eventID.String()+"/volunteer/roles", cache.VolunteerTTL, func(ctx context.Context) ([]dto.VolunteerRoleResponse, error) {
		roles, err := s.volunteers.ListRolesByEvent(ctx, eventID)
		if err != nil {
			return nil, err
		}
		out := make([]dto.VolunteerRoleResponse, 0, len(roles))
		for _, r := range roles {
			out = append(out, dto.NewVolunteerRoleResponse(r))
		}
		return out, nil
	})
	if err != nil {
		log.ErrorContext(ctx, "list volunteer roles", "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Volunteer Routes (Auth User)
// ---------------------------------------------------------------------------

// CreateVolunteerApplication handles POST /api/events/:id/volunteer/apply.
// Submits a volunteer application for the authenticated user. The
// request body selects one or more roles and may include a free-text
// message. The application starts in "pending" status until an
// organizer or volunteer admin reviews it.
func (s *Server) CreateVolunteerApplication(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.CreateVolunteerApplicationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	app, err := s.volunteers.Apply(ctx, u, eventID, req)
	if err != nil {
		if httpErr := volunteerApplyError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(ctx, "create volunteer application", "user_id", u.ID, "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	s.cache.Delete(ctx, "GET:/api/me/volunteer-applications:user:"+u.ID.String())
	s.cache.DeletePrefix(ctx, "GET:/api/me/volunteer-applications:user:"+u.ID.String()+"?")

	return c.JSON(http.StatusCreated, app)
}

// ListMyEventStaff handles GET /api/me/event-staff. Returns events where
// the authenticated user is an organizer or approved volunteer admin.
// Distinct from ListMyEvents which only returns owned events.
func (s *Server) ListMyEventStaff(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	ctx := c.Request().Context()
	key := "GET:/api/me/event-staff:user:" + u.ID.String()
	if q := c.Request().URL.RawQuery; q != "" {
		key += "?" + q
	}
	resp, err := cache.GetJSON(ctx, s.cache, key, cache.MyEventStaffTTL, func(ctx context.Context) (dto.EventStaffListResponse, error) {
		events, total, err := s.volunteers.ListStaffEventsByUser(ctx, u.ID, service.ListParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return dto.EventStaffListResponse{}, err
		}
		return dto.EventStaffListResponse{
			Events: events,
			Limit:  limit,
			Offset: offset,
			Total:  total,
		}, nil
	})
	if err != nil {
		log.ErrorContext(ctx, "list my event staff", "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ListMyVolunteerApplications handles GET /api/me/volunteer-applications.
// Cached per-user with MyVolunteerAppsTTL. The cache is invalidated
// when the user submits, cancels, or receives an approval/rejection on
// an application.
func (s *Server) ListMyVolunteerApplications(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	ctx := c.Request().Context()
	key := "GET:/api/me/volunteer-applications:user:" + u.ID.String()
	if q := c.Request().URL.RawQuery; q != "" {
		key += "?" + q
	}
	resp, err := cache.GetJSON(ctx, s.cache, key, cache.MyVolunteerAppsTTL, func(ctx context.Context) (dto.VolunteerApplicationListResponse, error) {
		apps, total, err := s.volunteers.ListByUser(ctx, u.ID, service.ListParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return dto.VolunteerApplicationListResponse{}, err
		}
		return dto.VolunteerApplicationListResponse{
			Applications: apps,
			Limit:        limit,
			Offset:       offset,
			Total:        total,
		}, nil
	})
	if err != nil {
		log.ErrorContext(ctx, "list my volunteer applications", "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// GetMyVolunteerApplication handles GET /api/volunteer-applications/:id.
// Scoped to the authenticated user: returns 404 for applications owned
// by a different user.
func (s *Server) GetMyVolunteerApplication(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	app, err := s.volunteers.GetByID(c.Request().Context(), appID, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrVolunteerAppNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "application not found")
		}
		log.ErrorContext(c.Request().Context(), "get volunteer application", "id", appID, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, app)
}

// CancelVolunteerApplication handles DELETE /api/volunteer-applications/:id.
// Only pending applications can be cancelled. Once approved or rejected,
// the organizer's action stands and the applicant cannot unilaterally
// withdraw.
func (s *Server) CancelVolunteerApplication(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	app, err := s.volunteers.Cancel(ctx, appID, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrVolunteerAppNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "application not found")
		}
		if errors.Is(err, service.ErrVolunteerInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only pending applications can be cancelled")
		}
		log.ErrorContext(ctx, "cancel volunteer application", "id", appID, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	s.cache.Delete(ctx, "GET:/api/me/volunteer-applications:user:"+u.ID.String())
	s.cache.DeletePrefix(ctx, "GET:/api/me/volunteer-applications:user:"+u.ID.String()+"?")

	return c.JSON(http.StatusOK, app)
}

// ---------------------------------------------------------------------------
// Organizer Routes
// ---------------------------------------------------------------------------

// UpsertVolunteerSettings handles PUT /api/events/:id/volunteer/settings.
// Creates or updates the volunteer settings for an event. Uses PUT
// (idempotent; same body produces the same result) rather than PATCH.
// Only the event organizer may update settings.
func (s *Server) UpsertVolunteerSettings(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.UpsertVolunteerSettingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	settings, err := s.volunteers.UpsertSettings(ctx, eventID, u.ID, req)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "upsert volunteer settings", "event_id", eventID, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.cache.Delete(ctx, "GET:/api/events/"+eventID.String()+"/volunteer/settings")

	return c.JSON(http.StatusOK, dto.NewVolunteerSettingsResponse(settings))
}

// CreateVolunteerRole handles POST /api/events/:id/volunteer/roles.
// Adds a new role to the event's volunteer role catalog. Role names
// must be unique per event (DB unique constraint enforces this).
func (s *Server) CreateVolunteerRole(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.CreateVolunteerRoleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	role, err := s.volunteers.CreateRole(ctx, eventID, u.ID, req)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrVolunteerRoleDuplicate) {
			return echo.NewHTTPError(http.StatusConflict, "role name already exists")
		}
		log.ErrorContext(ctx, "create volunteer role", "event_id", eventID, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.cache.Delete(ctx, "GET:/api/events/"+eventID.String()+"/volunteer/roles")

	return c.JSON(http.StatusCreated, dto.NewVolunteerRoleResponse(role))
}

// DeleteVolunteerRole handles DELETE /api/events/:id/volunteer/roles/:roleId.
// Removes a role from the catalog. Roles that already have applications
// attached cannot be deleted (ErrVolunteerRoleInUse) to preserve
// referential integrity and audit history.
func (s *Server) DeleteVolunteerRole(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	role, err := s.volunteers.DeleteRole(ctx, eventID, roleID, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrVolunteerRoleNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "role not found")
		}
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrVolunteerRoleInUse) {
			return echo.NewHTTPError(http.StatusConflict, "role has applications")
		}
		log.ErrorContext(ctx, "delete volunteer role", "event_id", eventID, "role_id", roleID, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.cache.Delete(ctx, "GET:/api/events/"+eventID.String()+"/volunteer/roles")

	return c.JSON(http.StatusOK, dto.NewVolunteerRoleResponse(role))
}

// ListEventVolunteerApplications handles GET /api/events/:id/volunteer/applications.
// Organizer/volunteer-admin view of all applications for the event.
// Not cached because these lists are reviewed interactively and must
// reflect real-time approval/rejection decisions.
func (s *Server) ListEventVolunteerApplications(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	apps, total, err := s.volunteers.ListByEvent(c.Request().Context(), eventID, u.ID, service.ListParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(c.Request().Context(), "list event volunteer applications", "event_id", eventID, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.VolunteerApplicationListResponse{
		Applications: apps,
		Limit:        limit,
		Offset:       offset,
		Total:        total,
	})
}

// ListEventVolunteers handles GET /api/events/:id/volunteers.
// Returns the confirmed volunteer roster for the event — the list of
// approved applications. Accessible by the organizer and approved
// volunteer admins.
func (s *Server) ListEventVolunteers(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	volunteers, total, err := s.volunteers.ListVolunteersByEvent(c.Request().Context(), eventID, u.ID, service.ListParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(c.Request().Context(), "list event volunteers", "event_id", eventID, "actor_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.VolunteerListResponse{
		Volunteers: volunteers,
		Limit:      limit,
		Offset:     offset,
		Total:      total,
	})
}

// GetVolunteerApplicationAsAdmin handles GET /api/volunteer-applications/:id/admin.
// Returns the full application including the applicant's identity and
// contact info. The user-facing GetMyVolunteerApplication returns a
// subset (no personal data of other users).
func (s *Server) GetVolunteerApplicationAsAdmin(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	app, err := s.volunteers.GetByIDAsAdmin(c.Request().Context(), appID, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrVolunteerAppNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "application not found")
		}
		log.ErrorContext(c.Request().Context(), "get volunteer application as admin", "id", appID, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, app)
}

// ApproveVolunteerApplication handles POST /api/volunteer-applications/:id/approve.
// Transitions a pending application to approved. The applicant's
// my/event-staff cache is invalidated so their dashboard immediately
// reflects the new staff status. Applications with admin-level roles
// can only be approved by the event organizer (not volunteer admins).
func (s *Server) ApproveVolunteerApplication(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	app, err := s.volunteers.Approve(ctx, appID, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrVolunteerAppNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "application not found")
		}
		if errors.Is(err, service.ErrVolunteerAdminRoleRestricted) {
			return echo.NewHTTPError(http.StatusForbidden, "only the organizer can approve admin-role volunteers")
		}
		if errors.Is(err, service.ErrVolunteerInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only pending applications can be approved")
		}
		log.ErrorContext(ctx, "approve volunteer application", "id", appID, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	s.cache.Delete(ctx, "GET:/api/me/volunteer-applications:user:"+app.UserID.String())
	s.cache.DeletePrefix(ctx, "GET:/api/me/volunteer-applications:user:"+app.UserID.String()+"?")
	s.cache.Delete(ctx, "GET:/api/me/event-staff:user:"+app.UserID.String())
	s.cache.DeletePrefix(ctx, "GET:/api/me/event-staff:user:"+app.UserID.String()+"?")

	return c.JSON(http.StatusOK, app)
}

// RejectVolunteerApplication handles POST /api/volunteer-applications/:id/reject.
// Transitions a pending application to rejected. Same admin-level role
// restriction as Approve. The applicant's application-list cache is
// invalidated so the rejection shows up promptly.
func (s *Server) RejectVolunteerApplication(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	ctx := c.Request().Context()
	app, err := s.volunteers.Reject(ctx, appID, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrVolunteerAppNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "application not found")
		}
		if errors.Is(err, service.ErrVolunteerAdminRoleRestricted) {
			return echo.NewHTTPError(http.StatusForbidden, "only the organizer can reject admin-role volunteers")
		}
		if errors.Is(err, service.ErrVolunteerInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only pending applications can be rejected")
		}
		log.ErrorContext(ctx, "reject volunteer application", "id", appID, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	s.cache.Delete(ctx, "GET:/api/me/volunteer-applications:user:"+app.UserID.String())
	s.cache.DeletePrefix(ctx, "GET:/api/me/volunteer-applications:user:"+app.UserID.String()+"?")

	return c.JSON(http.StatusOK, app)
}

// ---------------------------------------------------------------------------
// Error Helpers
// ---------------------------------------------------------------------------

func volunteerApplyError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "event not found")
	case errors.Is(err, service.ErrEventNotOpen):
		return echo.NewHTTPError(http.StatusConflict, "event is not open for applications")
	case errors.Is(err, service.ErrVolunteerNotOpen):
		return echo.NewHTTPError(http.StatusConflict, "volunteer applications are not open")
	case errors.Is(err, service.ErrVolunteerMaxReached):
		return echo.NewHTTPError(http.StatusConflict, "volunteer application cap reached")
	case errors.Is(err, service.ErrVolunteerDuplicate):
		return echo.NewHTTPError(http.StatusConflict, "already applied for this event")
	case errors.Is(err, service.ErrVolunteerRoleNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "role not found")
	default:
		return nil
	}
}
