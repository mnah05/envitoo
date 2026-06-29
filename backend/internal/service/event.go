package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/pulse/internal/db"
	"github.com/mnah05/pulse/internal/dto"
	"github.com/mnah05/pulse/internal/helper"
)

// EventService provides business logic for event operations.
type EventService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	log     *slog.Logger
}

// NewEventService creates an EventService with the given dependencies.
func NewEventService(pool *pgxpool.Pool, queries *db.Queries, log *slog.Logger) *EventService {
	return &EventService{pool: pool, queries: queries, log: log}
}

// ListParams controls pagination and filtering for list queries.
type ListParams struct {
	Limit            int32
	Offset           int32
	IncludeCancelled bool
}

// PublicListParams controls filtering for public event discovery.
type PublicListParams struct {
	Limit        int32
	Offset       int32
	Category     string
	VirtualMode  int32 // 0=all, 1=virtual, 2=physical
	Search       string
	StartsAfter  time.Time
	StartsBefore time.Time
}

// Create inserts a new event owned by organizerID. The slug is derived
// from the title — the caller never provides it. On slug collision the
// query is retried once with a UUID suffix appended to the slug.
func (s *EventService) Create(ctx context.Context, organizerID uuid.UUID, req dto.CreateEventRequest) (db.Event, error) {
	eventSlug, err := helper.Generate(req.Title)
	if err != nil {
		return db.Event{}, fmt.Errorf("slug generation failed: %w", err)
	}
	event := db.CreateEventParams{
		OrganizerID:      organizerID,
		Title:            req.Title,
		Description:      req.Description,
		StartsAt:         req.StartsAt,
		EndsAt:           req.EndsAt,
		Capacity:         req.Capacity,
		PriceInPaise:     req.PriceInPaise,
		Currency:         req.Currency,
		Status:           "published",
		RequiresApproval: req.RequiresApproval,
		CoverImageUrl:    req.CoverImageURL,
		Slug:             eventSlug,
		Category:         req.Category,
		IsVirtual:        req.IsVirtual,
		MeetingUrl:       req.MeetingURL,
		VenueName:        req.VenueName,
	}

	if req.CalendarID != nil {
		event.CalendarID = pgtype.UUID{Valid: true, Bytes: *req.CalendarID}
	}
	if req.Lat != nil {
		lat, err := helper.Float64ToNumeric(*req.Lat)
		if err != nil {
			return db.Event{}, fmt.Errorf("convert lat: %w", err)
		}
		event.Lat = lat
	}
	if req.Lng != nil {
		lng, err := helper.Float64ToNumeric(*req.Lng)
		if err != nil {
			return db.Event{}, fmt.Errorf("convert lng: %w", err)
		}
		event.Lng = lng
	}

	if req.IsVirtual {
		event.VenueName = nil
		event.Lat = pgtype.Numeric{Valid: false}
		event.Lng = pgtype.Numeric{Valid: false}
	} else {
		event.MeetingUrl = nil
	}

	created, err := s.queries.CreateEvent(ctx, event)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Slug collision — retry once with a UUID suffix.
			event.Slug = eventSlug + "-" + uuid.New().String()[:8]
			created, err = s.queries.CreateEvent(ctx, event)
			if err != nil {
				return db.Event{}, fmt.Errorf("create event after slug retry: %w", err)
			}
			return created, nil
		}
		return db.Event{}, fmt.Errorf("create event: %w", err)
	}
	return created, nil
}

// GetByID returns a single event by its UUID. Returns ErrEventNotFound if the
// event does not exist.
func (s *EventService) GetByID(ctx context.Context, id uuid.UUID) (db.Event, error) {
	event, err := s.queries.GetEventByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("get event by id: %w", err)
	}
	return event, nil
}

func isViewableEventStatus(status string) bool {
	return status == "published" || status == "cancelled" || status == "completed"
}

// GetViewableByID returns an event the public may view (published, cancelled,
// or completed). Returns ErrEventNotFound if the event does not exist or is
// not viewable.
func (s *EventService) GetViewableByID(ctx context.Context, id uuid.UUID) (db.Event, error) {
	event, err := s.GetByID(ctx, id)
	if err != nil {
		return db.Event{}, err
	}
	if !isViewableEventStatus(event.Status) {
		return db.Event{}, fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)
	}
	return event, nil
}

// GetViewableBySlug returns an event the public may view by slug.
func (s *EventService) GetViewableBySlug(ctx context.Context, slug string) (db.Event, error) {
	event, err := s.queries.GetEventBySlug(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event with slug %q not found: %w", slug, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("get event by slug: %w", err)
	}
	if !isViewableEventStatus(event.Status) {
		return db.Event{}, fmt.Errorf("event with slug %q not found: %w", slug, ErrEventNotFound)
	}
	return event, nil
}

// GetPublishedByID returns a published event by its UUID. Returns
// ErrEventNotFound if the event does not exist or is not published.
func (s *EventService) GetPublishedByID(ctx context.Context, id uuid.UUID) (db.Event, error) {
	event, err := s.queries.GetPublishedEventByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("get published event by id: %w", err)
	}
	return event, nil
}

// GetBySlug returns a published event by its slug. Returns ErrEventNotFound if
// the event does not exist or is not published.
func (s *EventService) GetBySlug(ctx context.Context, slug string) (db.Event, error) {
	event, err := s.queries.GetPublishedEventBySlug(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event with slug %q not found: %w", slug, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("get event by slug: %w", err)
	}
	return event, nil
}

// ListByOrganizer returns a paginated list of events owned by the given
// organizer, ordered by starts_at descending. The second return value is the
// total count matching the filter.
func (s *EventService) ListByOrganizer(ctx context.Context, organizerID uuid.UUID, params ListParams) ([]db.Event, int32, error) {
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

	events, err := s.queries.ListEventsByOrganizer(ctx, db.ListEventsByOrganizerParams{
		OrganizerID:      organizerID,
		Limit:            limit,
		Offset:           offset,
		IncludeCancelled: params.IncludeCancelled,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list events by organizer: %w", err)
	}

	total, err := s.queries.CountEventsByOrganizer(ctx, db.CountEventsByOrganizerParams{
		OrganizerID: organizerID,
		Column2:     params.IncludeCancelled,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count events by organizer: %w", err)
	}

	return events, total, nil
}

// ListPublic returns a paginated list of published events for public
// discovery, ordered by starts_at ascending. The second return value is the
// total count matching the filters.
func (s *EventService) ListPublic(ctx context.Context, params PublicListParams) ([]db.Event, int32, error) {
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

	startsAfter := params.StartsAfter
	if startsAfter.IsZero() {
		startsAfter = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	startsBefore := params.StartsBefore
	if startsBefore.IsZero() {
		startsBefore = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	listArgs := db.ListPublicEventsParams{
		Column1:    params.Category,
		Column2:    params.VirtualMode,
		Column3:    params.Search,
		StartsAt:   startsAfter,
		StartsAt_2: startsBefore,
		Limit:      limit,
		Offset:     offset,
	}

	events, err := s.queries.ListPublicEvents(ctx, listArgs)
	if err != nil {
		return nil, 0, fmt.Errorf("list public events: %w", err)
	}

	total, err := s.queries.CountPublicEvents(ctx, db.CountPublicEventsParams{
		Column1:    params.Category,
		Column2:    params.VirtualMode,
		Column3:    params.Search,
		StartsAt:   startsAfter,
		StartsAt_2: startsBefore,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count public events: %w", err)
	}

	return events, total, nil
}

// Update applies a partial patch to an existing event. Ownership is
// enforced in the SQL (WHERE id = $1 AND organizer_id = $18). If the
// event does not exist or belongs to a different organizer, the query
// returns pgx.ErrNoRows and this method returns ErrEventNotFound.
//
// Field-level validation is performed by the handler. Cross-field
// venue constraints are validated against the merged state here.
func (s *EventService) Update(ctx context.Context, id uuid.UUID, organizerID uuid.UUID, req dto.UpdateEventRequest) (db.Event, error) {
	current, err := s.queries.GetEventByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("get event for update: %w", err)
	}

	// Start with current values, override with non-nil request fields.
	params := db.UpdateEventParams{
		ID:               id,
		Description:      current.Description,
		StartsAt:         current.StartsAt,
		EndsAt:           current.EndsAt,
		Capacity:         current.Capacity,
		PriceInPaise:     current.PriceInPaise,
		Currency:         current.Currency,
		RequiresApproval: current.RequiresApproval,
		CoverImageUrl:    current.CoverImageUrl,
		CalendarID:       current.CalendarID,
		IsVirtual:        current.IsVirtual,
		MeetingUrl:       current.MeetingUrl,
		VenueName:        current.VenueName,
		Lat:              current.Lat,
		Lng:              current.Lng,
		Category:         current.Category,
		OrganizerID:      organizerID,
	}

	if req.Description != nil {
		params.Description = req.Description
	}
	if req.StartsAt != nil {
		params.StartsAt = *req.StartsAt
	}
	if req.EndsAt != nil {
		params.EndsAt = *req.EndsAt
	}
	if req.Capacity != nil {
		params.Capacity = *req.Capacity
	}
	if req.PriceInPaise != nil {
		params.PriceInPaise = *req.PriceInPaise
	}
	if req.Currency != nil {
		params.Currency = *req.Currency
	}
	if req.RequiresApproval != nil {
		params.RequiresApproval = *req.RequiresApproval
	}
	if req.CoverImageURL != nil {
		params.CoverImageUrl = req.CoverImageURL
	}
	if req.IsVirtual != nil {
		params.IsVirtual = *req.IsVirtual
	}
	if req.MeetingURL != nil {
		params.MeetingUrl = req.MeetingURL
	}
	if req.VenueName != nil {
		params.VenueName = req.VenueName
	}
	if req.Category != nil {
		params.Category = req.Category
	}
	if req.Lat != nil {
		lat, err := helper.Float64ToNumeric(*req.Lat)
		if err != nil {
			return db.Event{}, fmt.Errorf("convert lat: %w", err)
		}
		params.Lat = lat
	}
	if req.Lng != nil {
		lng, err := helper.Float64ToNumeric(*req.Lng)
		if err != nil {
			return db.Event{}, fmt.Errorf("convert lng: %w", err)
		}
		params.Lng = lng
	}

	// Validate merged state.
	if params.IsVirtual {
		if params.MeetingUrl == nil || *params.MeetingUrl == "" {
			return db.Event{}, fmt.Errorf("%w: meeting_url is required for virtual events", ErrInvalidEvent)
		}
	} else {
		if params.VenueName == nil || *params.VenueName == "" {
			return db.Event{}, fmt.Errorf("%w: venue_name is required for physical events", ErrInvalidEvent)
		}
		if !params.Lat.Valid {
			return db.Event{}, fmt.Errorf("%w: lat is required for physical events", ErrInvalidEvent)
		}
		if !params.Lng.Valid {
			return db.Event{}, fmt.Errorf("%w: lng is required for physical events", ErrInvalidEvent)
		}
	}
	if !params.EndsAt.After(params.StartsAt) {
		return db.Event{}, fmt.Errorf("%w: ends_at must be after starts_at", ErrInvalidEvent)
	}

	// Normalize virtual/physical fields.
	if params.IsVirtual {
		params.VenueName = nil
		params.Lat = pgtype.Numeric{Valid: false}
		params.Lng = pgtype.Numeric{Valid: false}
	} else {
		params.MeetingUrl = nil
	}

	event, err := s.queries.UpdateEvent(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("update event: %w", err)
	}
	return event, nil
}

// Cancel soft-cancels an event (sets status='cancelled'). Preserves all
// registrations, volunteers, kanban, and payment rows. The organizerID
// parameter enforces ownership in the SQL.
func (s *EventService) Cancel(ctx context.Context, id uuid.UUID, organizerID uuid.UUID) (db.Event, error) {
	event, err := s.queries.CancelEvent(ctx, db.CancelEventParams{
		ID:          id,
		OrganizerID: organizerID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Event{}, fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)
		}
		return db.Event{}, fmt.Errorf("cancel event: %w", err)
	}
	return event, nil
}
