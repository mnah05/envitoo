package dto

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateEventRequest is the request body for POST /api/events.
type CreateEventRequest struct {
	Title            string     `json:"title" validate:"required,min=1,max=200"`
	Description      *string    `json:"description" validate:"omitempty,max=5000"`
	StartsAt         time.Time  `json:"starts_at" validate:"required"`
	EndsAt           time.Time  `json:"ends_at" validate:"required,gtfield=StartsAt"`
	Capacity         int32      `json:"capacity" validate:"required,min=1"`
	PriceInPaise     int64      `json:"price_in_paise" validate:"min=0"`
	Currency         string     `json:"currency" validate:"required,len=3"`
	RequiresApproval bool       `json:"requires_approval"`
	CoverImageURL    *string    `json:"cover_image_url" validate:"omitempty,url"`
	CalendarID       *uuid.UUID `json:"calendar_id"`
	IsVirtual        bool       `json:"is_virtual"`
	MeetingURL       *string    `json:"meeting_url" validate:"omitempty,url"`
	VenueName        *string    `json:"venue_name" validate:"omitempty,max=200"`
	LocationAddress  *string    `json:"location_address" validate:"omitempty,max=500"`
	Lat              *float64   `json:"lat" validate:"omitempty,latitude"`
	Lng              *float64   `json:"lng" validate:"omitempty,longitude"`
	Category         *string    `json:"category" validate:"omitempty,oneof=technical cultural sports workshop seminar hackathon other"`
}

// UpdateEventRequest is the request body for PATCH /api/events/:id.
// PATCH semantics: partial update. Only send the fields you want to
// change. title, slug, status, organizer_id, created_at, and updated_at
// are immutable via this endpoint.
type UpdateEventRequest struct {
	Description      *string    `json:"description" validate:"omitempty,max=5000"`
	StartsAt         *time.Time `json:"starts_at"`
	EndsAt           *time.Time `json:"ends_at"`
	Capacity         *int32     `json:"capacity" validate:"omitempty,min=1"`
	PriceInPaise     *int64     `json:"price_in_paise" validate:"omitempty,min=0"`
	Currency         *string    `json:"currency" validate:"omitempty,len=3"`
	RequiresApproval *bool      `json:"requires_approval"`
	CoverImageURL    *string    `json:"cover_image_url" validate:"omitempty,url"`
	IsVirtual        *bool      `json:"is_virtual"`
	MeetingURL       *string    `json:"meeting_url" validate:"omitempty,url"`
	VenueName        *string    `json:"venue_name" validate:"omitempty,max=200"`
	LocationAddress  *string    `json:"location_address" validate:"omitempty,max=500"`
	Lat              *float64   `json:"lat" validate:"omitempty,latitude"`
	Lng              *float64   `json:"lng" validate:"omitempty,longitude"`
	Category         *string    `json:"category" validate:"omitempty,oneof=technical cultural sports workshop seminar hackathon other"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

type EventResponse struct {
	ID                uuid.UUID  `json:"id"`
	OrganizerID       uuid.UUID  `json:"organizer_id"`
	Title             string     `json:"title"`
	Description       *string    `json:"description,omitempty"`
	StartsAt          time.Time  `json:"starts_at"`
	EndsAt            time.Time  `json:"ends_at"`
	Capacity          int32      `json:"capacity"`
	RegistrationCount int32      `json:"registration_count"`
	PriceInPaise      int64      `json:"price_in_paise"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	Slug              string     `json:"slug"`
	Category          *string    `json:"category,omitempty"`
	RequiresApproval  bool       `json:"requires_approval"`
	CoverImageURL     *string    `json:"cover_image_url,omitempty"`
	CalendarID        *uuid.UUID `json:"calendar_id,omitempty"`
	IsVirtual         bool       `json:"is_virtual"`
	MeetingURL        *string    `json:"meeting_url,omitempty"`
	VenueName         *string    `json:"venue_name,omitempty"`
	LocationAddress   *string    `json:"location_address,omitempty"`
	Lat               *float64   `json:"lat,omitempty"`
	Lng               *float64   `json:"lng,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// EventListResponse wraps a paginated list of events with pagination
// metadata.
type EventListResponse struct {
	Events []EventResponse `json:"events"`
	Limit  int32           `json:"limit"`
	Offset int32           `json:"offset"`
	Total  int32           `json:"total"`
}

// NewEventResponse converts a db.Event row into an EventResponse.
func NewEventResponse(e db.Event) EventResponse {
	resp := EventResponse{
		ID:               e.ID,
		OrganizerID:      e.OrganizerID,
		Title:            e.Title,
		Description:      e.Description,
		StartsAt:         e.StartsAt,
		EndsAt:           e.EndsAt,
		Capacity:         e.Capacity,
		PriceInPaise:     e.PriceInPaise,
		Currency:         e.Currency,
		Status:           e.Status,
		Slug:             e.Slug,
		Category:         e.Category,
		RequiresApproval: e.RequiresApproval,
		CoverImageURL:    e.CoverImageUrl,
		IsVirtual:        e.IsVirtual,
		MeetingURL:       e.MeetingUrl,
		VenueName:        e.VenueName,
		LocationAddress:  e.LocationAddress,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}

	if e.CalendarID.Valid {
		id := uuid.UUID(e.CalendarID.Bytes)
		resp.CalendarID = &id
	}
	if e.Lat.Valid {
		f8, err := e.Lat.Float64Value()
		if err == nil {
			lat := f8.Float64
			resp.Lat = &lat
		}
	}
	if e.Lng.Valid {
		f8, err := e.Lng.Float64Value()
		if err == nil {
			lng := f8.Float64
			resp.Lng = &lng
		}
	}

	return resp
}

// NewEventListResponse converts a slice of EventResponse rows into a
// paginated EventListResponse.
func NewEventListResponse(events []EventResponse, limit, offset, total int32) EventListResponse {
	return EventListResponse{Events: events, Limit: limit, Offset: offset, Total: total}
}

// ToEventResponses converts a slice of db.Event to a slice of EventResponse.
func ToEventResponses(events []db.Event) []EventResponse {
	items := make([]EventResponse, 0, len(events))
	for _, e := range events {
		items = append(items, NewEventResponse(e))
	}
	return items
}

// ---------------------------------------------------------------------------
// Custom struct-level validations
// ---------------------------------------------------------------------------

// registerEventValidations registers struct-level validations for event
// request DTOs. These cover constraints that cannot be expressed with
// single-field `validate` tags — specifically the virtual/physical
// mutual-exclusion: a virtual event requires a meeting_url, a physical
// event requires venue_name + lat + lng.
func registerEventValidations(v *validator.Validate) {
	v.RegisterStructValidation(validateCreateEventRequest, CreateEventRequest{})
}

func validateCreateEventRequest(sl validator.StructLevel) {
	r := sl.Current().Interface().(CreateEventRequest)
	validateEventVenueConstraint(sl, r.IsVirtual, r.MeetingURL, r.VenueName, r.LocationAddress, r.Lat, r.Lng)
}

// validateEventVenueConstraint enforces:
//   - is_virtual=true  ⇒ meeting_url is required
//   - is_virtual=false ⇒ venue_name, location_address, lat, lng are required
func validateEventVenueConstraint(
	sl validator.StructLevel,
	isVirtual bool,
	meetingURL *string,
	venueName *string,
	locationAddress *string,
	lat *float64,
	lng *float64,
) {
	if isVirtual {
		if meetingURL == nil || *meetingURL == "" {
			sl.ReportError(meetingURL, "meeting_url", "meeting_url", "required_when_virtual", "")
		}
		return
	}
	if venueName == nil || *venueName == "" {
		sl.ReportError(venueName, "venue_name", "venue_name", "required_when_physical", "")
	}
	if locationAddress == nil || *locationAddress == "" {
		sl.ReportError(locationAddress, "location_address", "location_address", "required_when_physical", "")
	}
	if lat == nil {
		sl.ReportError(lat, "lat", "lat", "required_when_physical", "")
	}
	if lng == nil {
		sl.ReportError(lng, "lng", "lng", "required_when_physical", "")
	}
}
