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

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// CreateEvent handles POST /api/events. The handler follows a standard
// pattern: bind JSON → validate → load auth user → call service →
// invalidate cache → return response.
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

	ctx := c.Request().Context()
	event, err := s.events.Create(ctx, u.ID, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidEvent) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		log.ErrorContext(ctx, "create event", "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	// Invalidate the public event listing cache so newly created events
	// appear without waiting for the TTL to expire.
	s.cache.DeletePrefix(ctx, "GET:/api/events?")

	return c.JSON(http.StatusCreated, s.events.EventResponseBestEffort(ctx, event))
}

// GetEvent handles GET /api/events/:id. Uses cache.GetJSON to implement
// the cache-aside pattern: on cache miss, the callback populates the
// cache and the response is served from the returned value.
func (s *Server) GetEvent(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	ctx := c.Request().Context()
	resp, err := cache.GetJSON(ctx, s.cache, "GET:/api/events/"+id.String(), cache.EventTTL, func(ctx context.Context) (dto.EventResponse, error) {
		event, err := s.events.GetViewableByID(ctx, id)
		if err != nil {
			return dto.EventResponse{}, err
		}
		return s.events.EventResponseWithCount(ctx, event)
	})
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "get event", "id", id, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// GetEventBySlug handles GET /api/events/slug/:slug. Same pattern as
// GetEvent but looks up by slug (a human-readable URL path) instead
// of the internal UUID. Guests share the slug via event links.
func (s *Server) GetEventBySlug(c echo.Context) error {
	log := middleware.LoggerFromContext(c.Request().Context())

	slug := c.Param("slug")
	if slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "slug is required")
	}

	ctx := c.Request().Context()
	resp, err := cache.GetJSON(ctx, s.cache, "GET:/api/events/slug/"+slug, cache.EventTTL, func(ctx context.Context) (dto.EventResponse, error) {
		event, err := s.events.GetViewableBySlug(ctx, slug)
		if err != nil {
			return dto.EventResponse{}, err
		}
		return s.events.EventResponseWithCount(ctx, event)
	})
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "get event by slug", "slug", slug, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ListPublicEvents handles GET /api/events (public discovery). Supports
// optional query-parameter filters: limit, offset, is_virtual,
// starts_after, starts_before, category, and q (text search).
// The result is cached under a key derived from the full query string:
// two identical requests within PublicListTTL share a single DB query.
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
	category := c.QueryParam("category")
	search := c.QueryParam("q")
	ctx := c.Request().Context()

	listParams := service.PublicListParams{
		Limit:        limit,
		Offset:       offset,
		Category:     category,
		VirtualMode:  virtualMode,
		Search:       search,
		StartsAfter:  startsAfter,
		StartsBefore: startsBefore,
	}
	key := "GET:/api/events"
	if q := c.Request().URL.RawQuery; q != "" {
		key += "?" + q
	}
	resp, err := cache.GetJSON(ctx, s.cache, key, cache.PublicListTTL, func(ctx context.Context) (dto.EventListResponse, error) {
		events, total, err := s.events.ListPublic(ctx, listParams)
		if err != nil {
			return dto.EventListResponse{}, err
		}
		return dto.NewEventListResponse(events, limit, offset, total), nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.DebugContext(ctx, "client disconnected", "error", err)
			return nil
		}
		log.ErrorContext(ctx, "list public events", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, resp)
}

// ListMyEvents handles GET /api/me/events. Returns events owned by the
// authenticated user. Unlike the public listing, this is NOT cached
// (each request reads the DB directly) since organizers expect their
// own event list to reflect recent changes immediately.
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

	items := s.events.EventResponsesBestEffort(c.Request().Context(), events)
	return c.JSON(http.StatusOK, dto.NewEventListResponse(items, limit, offset, total))
}

// UpdateEvent handles PATCH /api/events/:id. Mapped to PATCH (partial
// update) rather than PUT (full replacement). The handler invalidates
// the cache for both the old slug (which may have changed) and the new
// slug so stale data is never served.
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

	ctx := c.Request().Context()
	oldSlug := s.lookupEventSlug(ctx, id)
	event, err := s.events.Update(ctx, id, u.ID, req)
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
		log.ErrorContext(ctx, "update event", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	keys := []string{"GET:/api/events/" + id.String()}
	if oldSlug != "" {
		keys = append(keys, "GET:/api/events/slug/"+oldSlug)
	}
	if event.Slug != "" {
		keys = append(keys, "GET:/api/events/slug/"+event.Slug)
	}
	s.cache.Delete(ctx, keys...)
	s.cache.DeletePrefix(ctx, "GET:/api/events?")

	return c.JSON(http.StatusOK, s.events.EventResponseBestEffort(ctx, event))
}

// CancelEvent handles DELETE /api/events/:id. Soft-deletes the event
// (sets status to cancelled; the row stays in the DB). A cancelled
// event is excluded from public listings but still accessible by
// direct ID for existing ticket holders.
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

	ctx := c.Request().Context()
	event, err := s.events.Cancel(ctx, id, u.ID)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "event not found")
		}
		log.ErrorContext(ctx, "cancel event", "id", id, "organizer_id", u.ID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	s.cache.Delete(ctx,
		"GET:/api/events/"+id.String(),
		"GET:/api/events/slug/"+event.Slug,
		"GET:/api/events/"+id.String()+"/registrations",
		"GET:/api/events/"+id.String()+"/volunteer/settings",
		"GET:/api/events/"+id.String()+"/volunteer/roles",
	)
	s.cache.DeletePrefix(ctx, "GET:/api/events?")
	s.cache.DeletePrefix(ctx, "GET:/api/events/"+id.String()+"/registrations?")

	return c.JSON(http.StatusOK, s.events.EventResponseBestEffort(ctx, event))
}

// validationError converts go-playground/validator ValidationErrors into
// a user-friendly 400 JSON response. Instead of "validation failed on
// field 'Title' on tag 'required'", it returns [{field, tag, reason}]
// where reason is a human sentence (see validationReason below).
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
// Used throughout the server to prevent unbounded result sets while
// allowing the caller to control page size up to a known ceiling.
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
// is empty, invalid, or negative. Combined with parseLimit to implement
// standard offset-based pagination.
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
// the zero time if the param is empty or invalid. The zero time is a
// convenient sentinel: filters like "starts_after" skip zero values
// instead of requiring a separate "unset" flag.
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
