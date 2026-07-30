package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

// ---------------------------------------------------------------------------
// Settings Request/Response
// ---------------------------------------------------------------------------

// Note 1: The volunteer feature has a three-tier model: Settings (per-event
// toggle), Roles (named positions like "barista" or "MC"), and Applications
// (user submissions to fill roles). Each tier has its own request/response
// DTO pair.

// UpsertVolunteerSettingsRequest is the request body for PUT /api/events/:id/volunteer/settings.
//
// Note 2: "Upsert" means INSERT or UPDATE — the service layer uses
// ON CONFLICT to create-or-replace in a single query.
type UpsertVolunteerSettingsRequest struct {
	Enabled         bool    `json:"enabled"`
	Description     *string `json:"description" validate:"omitempty,max=2000"`
	MaxApplications *int32  `json:"max_applications" validate:"omitempty,min=1"`
}

// VolunteerSettingsResponse is the JSON representation of volunteer settings.
type VolunteerSettingsResponse struct {
	EventID         uuid.UUID `json:"event_id"`
	Enabled         bool      `json:"enabled"`
	Description     *string   `json:"description,omitempty"`
	MaxApplications *int32    `json:"max_applications,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewVolunteerSettingsResponse converts a db.VolunteerSetting into a response DTO.
func NewVolunteerSettingsResponse(s db.VolunteerSetting) VolunteerSettingsResponse {
	return VolunteerSettingsResponse{
		EventID:         s.EventID,
		Enabled:         s.Enabled,
		Description:     s.Description,
		MaxApplications: s.MaxApplications,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Role Request/Response
// ---------------------------------------------------------------------------

// CreateVolunteerRoleRequest is the request body for POST /api/events/:id/volunteer/roles.
//
// Note 3: IsAdmin roles have elevated permissions like editing event details.
// Non-admin roles are for task-specific volunteers who cannot modify the event.
type CreateVolunteerRoleRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=100"`
	IsAdmin bool   `json:"is_admin"`
}

// VolunteerRoleResponse is the JSON representation of a volunteer role.
type VolunteerRoleResponse struct {
	ID        uuid.UUID `json:"id"`
	EventID   uuid.UUID `json:"event_id"`
	Name      string    `json:"name"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

// NewVolunteerRoleResponse converts a db.VolunteerRole into a response DTO.
func NewVolunteerRoleResponse(r db.VolunteerRole) VolunteerRoleResponse {
	return VolunteerRoleResponse{
		ID:        r.ID,
		EventID:   r.EventID,
		Name:      r.Name,
		IsAdmin:   r.IsAdmin,
		CreatedAt: r.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// Application Request/Response
// ---------------------------------------------------------------------------

// CreateVolunteerApplicationRequest is the request body for POST /api/events/:id/volunteer/apply.
//
// Note 4: Name and Email are optional because authenticated users fill these
// from their profile. Guest applicants would need to provide them explicitly,
// though the current flow requires authentication for volunteer applications.
type CreateVolunteerApplicationRequest struct {
	RoleID      uuid.UUID `json:"role_id" validate:"required"`
	Name        *string   `json:"name" validate:"omitempty,min=1,max=200"`
	Email       *string   `json:"email" validate:"omitempty,email"`
	Phone       *string   `json:"phone" validate:"omitempty,max=20"`
	Description string    `json:"description" validate:"required,min=1,max=5000"`
}

// VolunteerApplicationResponse is the JSON representation of a volunteer application.
//
// Note 5: RoleName is joined from the volunteer_roles table in the database
// query so the API client doesn't need a separate request to resolve role IDs
// to display names.
type VolunteerApplicationResponse struct {
	ID          uuid.UUID `json:"id"`
	EventID     uuid.UUID `json:"event_id"`
	UserID      uuid.UUID `json:"user_id"`
	RoleID      uuid.UUID `json:"role_id"`
	RoleName    string    `json:"role_name"`
	IsAdmin     bool      `json:"is_admin"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       *string   `json:"phone,omitempty"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// VolunteerApplicationListResponse wraps a paginated list of applications.
type VolunteerApplicationListResponse struct {
	Applications []VolunteerApplicationResponse `json:"applications"`
	Limit        int32                          `json:"limit"`
	Offset       int32                          `json:"offset"`
	Total        int32                          `json:"total"`
}

// EventStaffItemResponse is one event where the user is an approved volunteer admin.
//
// Note 6: This is used for the "my staff events" endpoint. It joins events,
// applications, and roles to show a volunteer admin which events they manage
// and what their role is called.
type EventStaffItemResponse struct {
	EventID       uuid.UUID `json:"event_id"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	StartsAt      time.Time `json:"starts_at"`
	EndsAt        time.Time `json:"ends_at"`
	EventStatus   string    `json:"event_status"`
	ApplicationID uuid.UUID `json:"application_id"`
	RoleID        uuid.UUID `json:"role_id"`
	RoleName      string    `json:"role_name"`
	IsAdmin       bool      `json:"is_admin"`
	ApprovedAt    time.Time `json:"approved_at"`
}

// EventStaffListResponse wraps a paginated list of staff events.
type EventStaffListResponse struct {
	Events []EventStaffItemResponse `json:"events"`
	Limit  int32                    `json:"limit"`
	Offset int32                    `json:"offset"`
	Total  int32                    `json:"total"`
}

// ---------------------------------------------------------------------------
// Volunteer Roster Response
// ---------------------------------------------------------------------------

// VolunteerResponse is the JSON representation of a confirmed volunteer
// from the volunteers roster table.
//
// Note 7: The volunteers roster is a materialized view (or join) that
// represents confirmed, active volunteers. It differs from applications
// — an application is pending until approved, then the volunteer appears
// in the roster.
type VolunteerResponse struct {
	ID        uuid.UUID `json:"id"`
	EventID   uuid.UUID `json:"event_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	Email     string    `json:"email"`
	Username  *string   `json:"username,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// VolunteerListResponse wraps a paginated list of confirmed volunteers.
type VolunteerListResponse struct {
	Volunteers []VolunteerResponse `json:"volunteers"`
	Limit      int32               `json:"limit"`
	Offset     int32               `json:"offset"`
	Total      int32               `json:"total"`
}
