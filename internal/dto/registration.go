package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/pulse/internal/db"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateRegistrationRequest is the request body for POST /api/events/:id/register.
type CreateRegistrationRequest struct {
	Name  *string `json:"name" validate:"omitempty,min=1,max=200"`
	Email *string `json:"email" validate:"omitempty,email"`
}

// CreateGuestRegistrationRequest is the request body for POST /api/events/:id/register/guest.
type CreateGuestRegistrationRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=200"`
	Email string `json:"email" validate:"required,email"`
}

// UpdateRegistrationStatusRequest is the request body for PATCH /api/registrations/:id/status.
type UpdateRegistrationStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending approved rejected cancelled"`
}

// CheckInRequest is the request body for POST /api/registrations/:id/check-in.
type CheckInRequest struct {
	AccessToken uuid.UUID `json:"access_token" validate:"required"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// RegistrationResponse is the JSON representation of a registration row.
type RegistrationResponse struct {
	ID             uuid.UUID  `json:"id"`
	EventID        uuid.UUID  `json:"event_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	AccessToken    uuid.UUID  `json:"access_token,omitempty"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
	CheckedInAt    *time.Time `json:"checked_in_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// RegistrationListResponse wraps a paginated list of registrations with pagination
// metadata.
type RegistrationListResponse struct {
	Registrations []RegistrationResponse `json:"registrations"`
	Limit         int32                  `json:"limit"`
	Offset        int32                  `json:"offset"`
	Total         int32                  `json:"total"`
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewRegistrationResponse converts a db.Registration row into a RegistrationResponse.
func NewRegistrationResponse(r db.Registration) RegistrationResponse {
	resp := RegistrationResponse{
		ID:          r.ID,
		EventID:     r.EventID,
		Name:        r.Name,
		Email:       r.Email,
		Status:      r.Status,
		AccessToken: r.AccessToken,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.UserID.Valid {
		uid := uuid.UUID(r.UserID.Bytes)
		resp.UserID = &uid
	}
	if r.TokenExpiresAt.Valid {
		t := r.TokenExpiresAt.Time
		resp.TokenExpiresAt = &t
	}
	if r.CheckedInAt.Valid {
		t := r.CheckedInAt.Time
		resp.CheckedInAt = &t
	}

	return resp
}

// NewRegistrationListResponse converts a slice of db.Registration rows into a
// paginated RegistrationListResponse.
func NewRegistrationListResponse(regs []db.Registration, limit, offset, total int32) RegistrationListResponse {
	items := make([]RegistrationResponse, 0, len(regs))
	for _, r := range regs {
		items = append(items, NewRegistrationResponse(r))
	}
	return RegistrationListResponse{
		Registrations: items,
		Limit:         limit,
		Offset:        offset,
		Total:         total,
	}
}
