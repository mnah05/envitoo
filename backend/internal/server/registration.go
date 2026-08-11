package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// ---------------------------------------------------------------------------
// Guest Routes (Public)
// ---------------------------------------------------------------------------

// RequestGuestOTP handles POST /api/events/:id/register/guest/otp.
// Generates a time-limited OTP and sends it via the email pipeline
// (NATS → mailer worker → Resend API). Each event/email pair has a
// configurable resend cooldown defined in the OTP config.
func (s *Server) RequestGuestOTP(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.RequestGuestOTPRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	ctx := c.Request().Context()
	resp, err := s.registrations.RequestGuestOTP(ctx, eventID, req)
	if err != nil {
		if httpErr := guestOTPError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(ctx, "request guest otp", "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// VerifyGuestOTP handles POST /api/events/:id/register/guest/otp/verify.
// Validates the HMAC-based OTP and returns a registration token that
// the guest uses to manage their registration (the token acts as their
// "session" since they have no Clerk JWT).
func (s *Server) VerifyGuestOTP(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.VerifyGuestOTPRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := s.validate.Struct(req); err != nil {
		return validationError(c, err)
	}

	ctx := c.Request().Context()
	resp, err := s.registrations.VerifyGuestOTP(ctx, eventID, req)
	if err != nil {
		if httpErr := guestOTPError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(ctx, "verify guest otp", "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

func guestOTPError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "event not found")
	case errors.Is(err, service.ErrEventNotOpen):
		return echo.NewHTTPError(http.StatusConflict, "event is not open for registration")
	case errors.Is(err, service.ErrOTPRateLimited):
		return echo.NewHTTPError(http.StatusTooManyRequests, "otp resend rate limited")
	case errors.Is(err, service.ErrOTPInvalid):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired otp")
	case errors.Is(err, service.ErrDependencyUnavailable):
		return echo.NewHTTPError(http.StatusServiceUnavailable, "service temporarily unavailable")
	default:
		return nil
	}
}

// CreateGuestRegistration handles POST /api/events/:id/register/guest.
// Creates a registration row linked to the guest's verified email (no
// local db.User — user_id IS NULL). The response includes an access
// token used for subsequent GET/DELETE operations on this registration.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.CreateGuest(ctx, eventID, req)
	if err != nil {
		if httpErr := registrationCreateError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(ctx, "create guest registration", "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	resp := dto.NewRegistrationResponse(reg)
	token := reg.AccessToken
	resp.AccessToken = &token
	return c.JSON(http.StatusCreated, resp)
}

// GetGuestRegistration handles GET /api/guest-registrations?token=.
// Guests authenticate via a UUID query parameter rather than a session
// cookie or JWT header — this is a bearer token approach designed for
// the email-link flow where no traditional auth mechanism exists.
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

	ctx := c.Request().Context()
	resp, err := cache.GetJSON(ctx, s.cache, "GET:/api/guest-registrations?token="+token.String(), cache.GuestTicketTTL, func(ctx context.Context) (dto.RegistrationResponse, error) {
		reg, err := s.registrations.GetByAccessToken(ctx, token)
		if err != nil {
			return dto.RegistrationResponse{}, err
		}
		return dto.NewRegistrationResponse(reg), nil
	})
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(ctx, "get guest registration", "token", token, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// CancelGuestRegistration handles DELETE /api/guest-registrations?token=.
// Soft-deletes the guest registration and returns the cancelled row.
// Cache invalidation ensures the event's ticket count is updated for
// subsequent requests.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.CancelGuest(ctx, token)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(ctx, "cancel guest registration", "token", token, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// ---------------------------------------------------------------------------
// User Routes (Protected)
// ---------------------------------------------------------------------------

// CreateRegistration handles POST /api/events/:id/register.
// Authenticated-user variant of the guest flow. The user's identity is
// taken from the Clerk JWT (via LoadUser middleware) rather than an
// email OTP flow.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.Create(ctx, u, eventID, req)
	if err != nil {
		if httpErr := registrationCreateError(err); httpErr != nil {
			return httpErr
		}
		log.ErrorContext(ctx, "create registration", "user_id", u.ID, "event_id", eventID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	resp := dto.NewRegistrationResponse(reg)
	token := reg.AccessToken
	resp.AccessToken = &token
	return c.JSON(http.StatusCreated, resp)
}

// GetRegistration handles GET /api/registrations/:id. Scoped to the
// authenticated user: the service layer filters by user_id so one
// user cannot view another's registration.
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

// CancelRegistration handles DELETE /api/registrations/:id. Only the
// registration owner (user_id match) can cancel. An organizer/admin
// should use RejectRegistration instead.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.Cancel(ctx, id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		log.ErrorContext(ctx, "cancel registration", "id", id, "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// ListMyRegistrations handles GET /api/me/registrations. Cached per-user
// so that repeated fetches from the frontend navigation do not hit the
// DB each time. The cache is invalidated on any registration create,
// cancel, or status change via forgetRegistration.
func (s *Server) ListMyRegistrations(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	ctx := c.Request().Context()
	key := "GET:/api/me/registrations:user:" + u.ID.String()
	if q := c.Request().URL.RawQuery; q != "" {
		key += "?" + q
	}
	resp, err := cache.GetJSON(ctx, s.cache, key, cache.MyRegistrationsTTL, func(ctx context.Context) (dto.RegistrationListResponse, error) {
		regs, total, err := s.registrations.ListByUser(ctx, u.ID, service.ListParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return dto.RegistrationListResponse{}, err
		}
		return dto.NewRegistrationListResponse(regs, limit, offset, total), nil
	})
	if err != nil {
		log.ErrorContext(ctx, "list my registrations", "user_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Admin Routes (Organizer Only)
// ---------------------------------------------------------------------------

// ListEventRegistrations handles GET /api/events/:id/registrations.
// Returns the attendee roster for an event. Access is gated by
// EnsureEventStaff — the organizer or an approved volunteer admin may
// view this list. Authorization is checked BEFORE the cache lookup to
// prevent a non-staff caller from receiving a cached staff response.
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

	ctx := c.Request().Context()
	// Authorize before any cache hit. The shared key intentionally serves
	// the same staff payload to organizer and volunteer admins; without a
	// pre-check, a non-staff caller would receive a cached staff response.
	if err := s.registrations.EnsureEventStaff(ctx, eventID, u.ID); err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "list event registrations auth", "event_id", eventID, "actor_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	key := "GET:/api/events/" + eventID.String() + "/registrations"
	if q := c.Request().URL.RawQuery; q != "" {
		key += "?" + q
	}
	resp, err := cache.GetJSON(ctx, s.cache, key, cache.EventRegistrationsTTL, func(ctx context.Context) (dto.RegistrationListResponse, error) {
		regs, total, err := s.registrations.ListByEvent(ctx, eventID, u.ID, service.ListParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return dto.RegistrationListResponse{}, err
		}
		return dto.NewRegistrationListResponse(regs, limit, offset, total), nil
	})
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "list event registrations", "event_id", eventID, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// GetRegistrationAsAdmin handles GET /api/registrations/:id/admin.
// Returns full registration details for an organizer/volunteer-admin,
// including the attendee's identity that the user-facing endpoint
// deliberately omits.
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
// Transitions a pending registration to approved. This is the organizer's
// explicit confirmation — the attendee receives a spot and can later be
// checked in. If the event has reached its capacity, ErrEventFull is
// returned and no change is made.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.Approve(ctx, id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrEventStaffForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "registration approval is not allowed for this event")
		}
		if errors.Is(err, service.ErrEventFull) {
			return echo.NewHTTPError(http.StatusConflict, "event is full")
		}
		if errors.Is(err, service.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, "registration status conflict")
		}
		log.ErrorContext(ctx, "approve registration", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// RejectRegistration handles POST /api/registrations/:id/reject.
// Transitions a pending registration to rejected. Once rejected, the
// registration cannot be re-approved; the attendee must re-apply.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.Reject(ctx, id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrEventStaffForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "registration rejection is not allowed for this event")
		}
		if errors.Is(err, service.ErrRegistrationInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only pending registrations can be rejected")
		}
		if errors.Is(err, service.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, "registration status conflict")
		}
		log.ErrorContext(ctx, "reject registration", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// UpdateRegistrationStatus handles PATCH /api/registrations/:id/status.
// A flexible status endpoint for bulk or programmatic transitions.
// The allowed transitions are validated by the service layer.
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

	ctx := c.Request().Context()
	reg, err := s.registrations.UpdateStatus(ctx, id, u.ID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrEventFull) {
			return echo.NewHTTPError(http.StatusConflict, "event is full")
		}
		if errors.Is(err, service.ErrRegistrationInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "registration status transition not allowed")
		}
		if errors.Is(err, service.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, "registration status conflict")
		}
		log.ErrorContext(ctx, "update registration status", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// CheckInAttendee handles POST /api/registrations/:id/check-in.
// Marks an approved registration as checked in at the event door.
// The access_token in the request body is validated against the stored
// token; for organizers checking in via the dashboard the token is
// optional (they have event-level access already).
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

	ctx := c.Request().Context()
	reg, err := s.registrations.CheckIn(ctx, id, u.ID, req.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrRegistrationInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only approved registrations can be checked in")
		}
		if errors.Is(err, service.ErrEventNotOpen) {
			return echo.NewHTTPError(http.StatusConflict, "check-in is not allowed for this event")
		}
		log.ErrorContext(ctx, "check in attendee", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

	return c.JSON(http.StatusOK, dto.NewRegistrationResponse(reg))
}

// CheckInByEventToken handles POST /api/events/:id/check-in. Scans
// registrations by event ID + access token without requiring the
// registration ID. Convenient for QR-code scanners where the event
// context is known but the specific registration ID is not.
func (s *Server) CheckInByEventToken(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
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

	ctx := c.Request().Context()
	reg, err := s.registrations.CheckInByEventToken(ctx, eventID, u.ID, req.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "registration not found")
		}
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrRegistrationInvalidStatus) {
			return echo.NewHTTPError(http.StatusConflict, "only approved registrations can be checked in")
		}
		if errors.Is(err, service.ErrEventNotOpen) {
			return echo.NewHTTPError(http.StatusConflict, "check-in is not allowed for this event")
		}
		log.ErrorContext(ctx, "check in by event token", "event_id", eventID, "actor_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.forgetRegistration(ctx, reg)

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
	case errors.Is(err, service.ErrGuestEmailNotVerified):
		return echo.NewHTTPError(http.StatusForbidden, "email not verified")
	case errors.Is(err, service.ErrDependencyUnavailable):
		return echo.NewHTTPError(http.StatusServiceUnavailable, "service temporarily unavailable")
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Cache invalidation helpers
// ---------------------------------------------------------------------------

// forgetRegistration deletes all cache keys that might be affected by
// a change to the given registration. This includes the direct
// guest-registration token key, the event detail page, the event's
// registration list (including its page-variant prefix), and the
// user's my-registrations key.
// Needs a DB slug lookup before it can delete slug-based keys.
func (s *Server) forgetRegistration(ctx context.Context, reg db.Registration) {
	if s.cache == nil {
		return
	}
	keys := []string{
		"GET:/api/guest-registrations?token=" + reg.AccessToken.String(),
		"GET:/api/events/" + reg.EventID.String(),
		"GET:/api/events/" + reg.EventID.String() + "/registrations",
	}
	if reg.UserID.Valid {
		uid := uuid.UUID(reg.UserID.Bytes)
		keys = append(keys, "GET:/api/me/registrations:user:"+uid.String())
	}
	if slug := s.lookupEventSlug(ctx, reg.EventID); slug != "" {
		keys = append(keys, "GET:/api/events/slug/"+slug)
	}
	s.cache.Delete(ctx, keys...)
	s.cache.DeletePrefix(ctx, "GET:/api/events?")
	s.cache.DeletePrefix(ctx, "GET:/api/events/"+reg.EventID.String()+"/registrations?")
	if reg.UserID.Valid {
		uid := uuid.UUID(reg.UserID.Bytes)
		s.cache.DeletePrefix(ctx, "GET:/api/me/registrations:user:"+uid.String()+"?")
	}
}

// forgetRegistrationsByEmail looks up every registration for an email
// and invalidates each one. This is the callback passed to LoadUser as
// GuestRegistrationInvalidator. When a guest authenticates for the
// first time, their guest registrations are linked to the new local
// user; this function clears the stale cache entries so the user's
// dashboard immediately shows their newly-linked tickets.
// userID is the local user who received the linked rows; their
// me/registrations cache must be dropped so newly linked tickets appear.
func (s *Server) forgetRegistrationsByEmail(ctx context.Context, userID uuid.UUID, email string) {
	if s.cache == nil || email == "" {
		return
	}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT r.event_id, r.access_token, e.slug
		FROM registrations r
		JOIN events e ON e.id = r.event_id
		WHERE r.email = $1
	`, email)
	if err != nil {
		s.log.WarnContext(ctx, "cache invalidation lookup failed", "email", email, "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var eventID, token uuid.UUID
		var slug string
		if err := rows.Scan(&eventID, &token, &slug); err != nil {
			s.log.WarnContext(ctx, "cache invalidation scan failed", "error", err)
			return
		}
		keys := []string{
			"GET:/api/guest-registrations?token=" + token.String(),
			"GET:/api/events/" + eventID.String(),
			"GET:/api/events/" + eventID.String() + "/registrations",
		}
		if slug != "" {
			keys = append(keys, "GET:/api/events/slug/"+slug)
		}
		s.cache.Delete(ctx, keys...)
		s.cache.DeletePrefix(ctx, "GET:/api/events?")
		s.cache.DeletePrefix(ctx, "GET:/api/events/"+eventID.String()+"/registrations?")
	}
	_ = rows.Err()

	if userID != uuid.Nil {
		s.cache.Delete(ctx, "GET:/api/me/registrations:user:"+userID.String())
		s.cache.DeletePrefix(ctx, "GET:/api/me/registrations:user:"+userID.String()+"?")
	}
}

// lookupEventSlug is a cache-invalidation helper that fetches a single
// event's slug from the DB. The result is NOT cached because this is
// only called on mutation paths where the cache is being invalidated anyway.
func (s *Server) lookupEventSlug(ctx context.Context, eventID uuid.UUID) string {
	event, err := s.queries.GetEventByID(ctx, eventID)
	if err != nil {
		return ""
	}
	return event.Slug
}
