package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// CreateEvent handles POST /api/events.
func (s *Server) CreateEvent(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	var req dto.CreateEventRequest
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

	event, err := s.events.Create(c.Request().Context(), u.ID, req)
	if err != nil {
		log.ErrorContext(c.Request().Context(), "create event", "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, dto.NewEventResponse(event))
}

// GetEvent handles GET /api/events/:id.
func (s *Server) GetEvent(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	event, err := s.events.GetViewableByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(c.Request().Context(), "get event", "id", id, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	resp, err := s.events.EventResponseWithCount(c.Request().Context(), event)
	if err != nil {
		log.ErrorContext(c.Request().Context(), "get event registration count", "id", id, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// GetEventBySlug handles GET /api/events/slug/:slug.
func (s *Server) GetEventBySlug(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	slug := c.Param("slug")
	if slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "slug is required")
	}

	event, err := s.events.GetViewableBySlug(c.Request().Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(c.Request().Context(), "get event by slug", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	resp, err := s.events.EventResponseWithCount(c.Request().Context(), event)
	if err != nil {
		log.ErrorContext(c.Request().Context(), "get event by slug registration count", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ListPublicEvents handles GET /api/events (public discovery).
func (s *Server) ListPublicEvents(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	var virtualMode int32
	switch c.QueryParam("is_virtual") {
	case "true":
		virtualMode = 1
	case "false":
		virtualMode = 2
	default:
		virtualMode = 0
	}

	startsAfter := parseTime(c.QueryParam("starts_after"))
	startsBefore := parseTime(c.QueryParam("starts_before"))

	events, total, err := s.events.ListPublic(c.Request().Context(), service.PublicListParams{
		Limit:        limit,
		Offset:       offset,
		Category:     c.QueryParam("category"),
		VirtualMode:  virtualMode,
		Search:       c.QueryParam("q"),
		StartsAfter:  startsAfter,
		StartsBefore: startsBefore,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.DebugContext(c.Request().Context(), "client disconnected", "error", err)
			return nil
		}
		log.ErrorContext(c.Request().Context(), "list public events", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewEventListResponse(events, limit, offset, total))
}

// ListMyEvents handles GET /api/me/events.
func (s *Server) ListMyEvents(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	limit := parseLimit(c.QueryParam("limit"), 20, 100)
	offset := parseOffset(c.QueryParam("offset"))

	events, total, err := s.events.ListByOrganizer(c.Request().Context(), u.ID, service.ListParams{
		Limit:            limit,
		Offset:           offset,
		IncludeCancelled: c.QueryParam("include_cancelled") == "true",
	})
	if err != nil {
		log.ErrorContext(c.Request().Context(), "list my events", "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	items, err := s.events.EventResponsesWithCounts(c.Request().Context(), events)
	if err != nil {
		log.ErrorContext(c.Request().Context(), "list my events registration counts", "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewEventListResponse(items, limit, offset, total))
}

// UpdateEvent handles PATCH /api/events/:id.
func (s *Server) UpdateEvent(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.UpdateEventRequest
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

	event, err := s.events.Update(c.Request().Context(), id, u.ID, req)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		if errors.Is(err, service.ErrEventStaffForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, "insufficient event permissions")
		}
		if errors.Is(err, service.ErrInvalidEvent) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		log.ErrorContext(c.Request().Context(), "update event", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewEventResponse(event))
}

// CancelEvent handles DELETE /api/events/:id.
func (s *Server) CancelEvent(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	event, err := s.events.Cancel(c.Request().Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(c.Request().Context(), "cancel event", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.NewEventResponse(event))
}

// validationError converts a validator.ValidationErrors into a 400
// response with a structured error body. Each field error includes the
// field name, the tag that failed, and the tag's param if any.
func validationError(c echo.Context, err error) error {
	var valErrs validator.ValidationErrors
	if !errors.As(err, &valErrs) {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	type fieldError struct {
		Field  string `json:"field"`
		Tag    string `json:"tag"`
		Value  string `json:"value,omitempty"`
		Param  string `json:"param,omitempty"`
		Reason string `json:"reason"`
	}

	errs := make([]fieldError, 0, len(valErrs))
	for _, fe := range valErrs {
		errs = append(errs, fieldError{
			Field:  fe.Field(),
			Tag:    fe.Tag(),
			Value:  stringify(fe.Value()),
			Param:  fe.Param(),
			Reason: validationReason(fe),
		})
	}

	return c.JSON(http.StatusBadRequest, map[string]any{
		"message": "validation failed",
		"errors":  errs,
	})
}

// validationReason produces a human-readable explanation for a single
// validator field error. Extend this as new validation tags are added.
func validationReason(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "min":
		return "value must be at least " + fe.Param()
	case "max":
		return "value must be at most " + fe.Param()
	case "len":
		return "value must be exactly " + fe.Param() + " characters"
	case "gtfield":
		return "must be after " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	case "url":
		return "must be a valid URL"
	case "latitude":
		return "must be a valid latitude (-90 to 90)"
	case "longitude":
		return "must be a valid longitude (-180 to 180)"
	case "required_when_virtual":
		return "meeting_url is required for virtual events"
	case "required_when_physical":
		return "this field is required for physical events"
	default:
		return "validation failed on tag '" + fe.Tag() + "'"
	}
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	return formatValue(v)
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int32, int64:
		return strconv.FormatInt(toInt64(val), 10)
	default:
		return ""
	}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	}
	return 0
}

// parseLimit parses a limit query parameter, applying a default and a
// maximum cap. Returns the default if the param is empty or invalid.
func parseLimit(raw string, defaultVal, max int32) int32 {
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultVal
	}
	if n > int(max) {
		return max
	}
	return int32(n)
}

// parseOffset parses an offset query parameter. Returns 0 if the param
// is empty, invalid, or negative.
func parseOffset(raw string) int32 {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return int32(n)
}

// parseTime parses an ISO 8601 timestamp from a query parameter. Returns
// the zero time if the param is empty or invalid.
// Handles both RFC3339 (no fractional seconds) and RFC3339Nano (with
// fractional seconds), e.g. "2026-06-29T11:09:08Z" or "2026-06-29T11:09:08.875Z".
func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}
