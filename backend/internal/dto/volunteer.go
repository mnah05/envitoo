package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

// ---------------------------------------------------------------------------
// Settings Request/Response
// ---------------------------------------------------------------------------

// UpsertVolunteerSettingsRequest is the request body for PUT /api/events/:id/volunteer/settings.
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
type CreateVolunteerApplicationRequest struct {
	RoleID      uuid.UUID `json:"role_id" validate:"required"`
	Name        *string   `json:"name" validate:"omitempty,min=1,max=200"`
	Email       *string   `json:"email" validate:"omitempty,email"`
	Phone       *string   `json:"phone" validate:"omitempty,max=20"`
	Description string    `json:"description" validate:"required,min=1,max=5000"`
}

// VolunteerApplicationResponse is the JSON representation of a volunteer application.
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
