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

// Note 1: DTOs (Data Transfer Objects) are the explicit shape of data over
// the wire. They decouple the API contract from internal database models so
// changes to one do not silently break the other.

// CreateEventRequest is the request body for POST /api/events.
//
// Note 2: Validate tags use the go-playground/validator library. Each tag
// expresses a constraint: "required" means the field must be non-zero,
// "min=1,max=200" constrains string length, "gtfield=StartsAt" ensures
// EndsAt is strictly after StartsAt, and "oneof=..." restricts to an
// enumerated set of allowed values. Pointer fields (*string, *float64)
// use "omitempty" so the validator skips them if nil.
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
	Visibility       *string    `json:"visibility" validate:"omitempty,oneof=public unlisted"`
}

// UpdateEventRequest is the request body for PATCH /api/events/:id.
// PATCH semantics: partial update. Only send the fields you want to
// change. title, slug, status, organizer_id, created_at, and updated_at
// are immutable via this endpoint.
//
// Note 3: Every field is a pointer (*string, *time.Time, *int32). This
// distinguishes "not sent" (nil) from "send zero value" (e.g. setting
// price_in_paise to 0). For PATCH endpoints this is critical — without
// pointers you cannot tell whether the client wants to clear a field or
// simply omitted it.
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
	Visibility       *string    `json:"visibility" validate:"omitempty,oneof=public unlisted"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// Note 4: Response structs use `json:",omitempty"` on pointer fields so
// null values are excluded from the JSON output. This keeps responses
// clean — a physical event won't include "meeting_url": null.

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
	Visibility        string     `json:"visibility"`
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
	Warning           *string    `json:"warning,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// EventListResponse wraps a paginated list of events with pagination
// metadata.
//
// Note 5: The pagination fields (limit, offset, total) are included in every
// list response so the client knows what page it's on and how many results
// exist. This is essential for building paginated UIs — the frontend uses
// total + limit to calculate page count.
type EventListResponse struct {
	Events []EventResponse `json:"events"`
	Limit  int32           `json:"limit"`
	Offset int32           `json:"offset"`
	Total  int32           `json:"total"`
}

// NewEventResponse converts a db.Event row into an EventResponse.
//
// Note 6: sqlc-generated db.Event uses pgtype types (like pgtype.UUID,
// pgtype.Numeric) which have a .Valid field to distinguish null from zero.
// The DTO layer converts these to standard Go types (*uuid.UUID, *float64)
// so API consumers get clean JSON with proper null handling.
//
// Note 7: CalendarID and lat/lng use conditional conversion because they are
// nullable database columns. The .Bytes field on pgtype.UUID provides raw
// bytes that can be cast directly to uuid.UUID when Valid is true.
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
//
// Note 8: The caller (server handler) passes limit, offset, and total from
// the database query result. This function simply wraps them into the
// response shape — it doesn't perform any pagination logic itself.
func NewEventListResponse(events []EventResponse, limit, offset, total int32) EventListResponse {
	return EventListResponse{Events: events, Limit: limit, Offset: offset, Total: total}
}

// ToEventResponses converts a slice of db.Event to a slice of EventResponse.
//
// Note 9: The slice is pre-allocated with len(events) capacity to avoid
// multiple append-driven reallocations. This is a common Go performance
// pattern when the final size is known in advance.
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
