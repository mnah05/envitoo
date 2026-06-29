package server

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// ---------------------------------------------------------------------------
// Guest Routes (Public)
// ---------------------------------------------------------------------------

// CreateGuestRegistration handles POST /api/events/:id/register/guest.
func (s *Server) CreateGuestRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.CreateGuestRegistrationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	reg, err := s.registrations.CreateGuest(c.Request().Context(), eventID, req)
	if err != nil {
		if httpErr := registrationCreateError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(c.Request().Context(), "create guest registration", "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, dto.NewRegistrationResponse(reg))
}

// GetGuestRegistration handles GET /api/guest-registrations?token=.
func (s *Server) GetGuestRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	tokenStr := c.QueryParam("token")
	if tokenStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid token")
	}

	reg, err := s.registrations.GetByAccessToken(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(c.Request().Context(), "get guest registration", "token", token, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// CancelGuestRegistration handles DELETE /api/guest-registrations?token=.
func (s *Server) CancelGuestRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	tokenStr := c.QueryParam("token")
	if tokenStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid token")
	}

	reg, err := s.registrations.CancelGuest(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(c.Request().Context(), "cancel guest registration", "token", token, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// ---------------------------------------------------------------------------
// User Routes (Protected)
// ---------------------------------------------------------------------------

// CreateRegistration handles POST /api/events/:id/register.
func (s *Server) CreateRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.CreateRegistrationRequest
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

	reg, err := s.registrations.Create(c.Request().Context(), u, eventID, req)
	if err != nil {
		if httpErr := registrationCreateError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(c.Request().Context(), "create registration", "user_id", u.ID, "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, dto.NewRegistrationResponse(reg))
}

// GetRegistration handles GET /api/registrations/:id.
func (s *Server) GetRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	reg, err := s.registrations.GetByID(c.Request().Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(c.Request().Context(), "get registration", "id", id, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// CancelRegistration handles DELETE /api/registrations/:id.
func (s *Server) CancelRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	reg, err := s.registrations.Cancel(c.Request().Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(c.Request().Context(), "cancel registration", "id", id, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// ListMyRegistrations handles GET /api/me/registrations.
func (s *Server) ListMyRegistrations(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	regs, total, err := s.registrations.ListByUser(c.Request().Context(), u.ID, service.ListParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.ErrorContext(c.Request().Context(), "list my registrations", "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationListResponse(regs, limit, offset, total))
}

// ---------------------------------------------------------------------------
// Admin Routes (Organizer Only)
// ---------------------------------------------------------------------------

// ListEventRegistrations handles GET /api/events/:id/registrations.
func (s *Server) ListEventRegistrations(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 500)
	offset := parseOffset(c.QueryParam("offset"))

	regs, total, err := s.registrations.ListByEvent(c.Request().Context(), eventID, u.ID, service.ListParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(c.Request().Context(), "list event registrations", "event_id", eventID, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationListResponse(regs, limit, offset, total))
}

// GetRegistrationAsAdmin handles GET /api/registrations/:id/admin.
func (s *Server) GetRegistrationAsAdmin(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	reg, err := s.registrations.GetByIDAsAdmin(c.Request().Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(c.Request().Context(), "get registration as admin", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// ApproveRegistration handles POST /api/registrations/:id/approve.
func (s *Server) ApproveRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	reg, err := s.registrations.Approve(c.Request().Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventStaffForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "registration approval is not allowed for this event")
		}
		if errors.Is(err, service.ErrEventFull) {
			return echo.NewHTTPError(http.StatusConflict, "event is full")
		}
		log.ErrorContext(c.Request().Context(), "approve registration", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// RejectRegistration handles POST /api/registrations/:id/reject.
func (s *Server) RejectRegistration(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	reg, err := s.registrations.Reject(c.Request().Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventStaffForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "registration rejection is not allowed for this event")
		}
		if errors.Is(err, service.ErrRegistrationInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only pending registrations can be rejected")
		}
		log.ErrorContext(c.Request().Context(), "reject registration", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// UpdateRegistrationStatus handles PATCH /api/registrations/:id/status.
func (s *Server) UpdateRegistrationStatus(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	var req dto.UpdateRegistrationStatusRequest
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

	reg, err := s.registrations.UpdateStatus(c.Request().Context(), id, u.ID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventFull) {
			return echo.NewHTTPError(http.StatusConflict, "event is full")
		}
		log.ErrorContext(c.Request().Context(), "update registration status", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// CheckInAttendee handles POST /api/registrations/:id/check-in.
func (s *Server) CheckInAttendee(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid registration id")
	}

	var req dto.CheckInRequest
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

	reg, err := s.registrations.CheckIn(c.Request().Context(), id, u.ID, req.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventNotOpen) {
			return echo.NewHTTPError(http.StatusConflict, "check-in is not allowed for this event")
		}
		log.ErrorContext(c.Request().Context(), "check in attendee", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

func registrationCreateError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "event not found")
	case errors.Is(err, service.ErrConflict):
		return echo.NewHTTPError(http.StatusConflict, "already registered for this event")
	case errors.Is(err, service.ErrEventFull):
		return echo.NewHTTPError(http.StatusConflict, "event is full")
	case errors.Is(err, service.ErrEventNotOpen):
		return echo.NewHTTPError(http.StatusConflict, "event is not open for registration")
	default:
		return nil
	}
}
