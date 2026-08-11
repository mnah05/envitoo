package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// Note 1: Registration has two flows: authenticated (logged-in user) and
// guest (email + OTP). The authenticated flow pulls the user's email from
// the session, while the guest flow collects name + email + OTP verification.

// CreateRegistrationRequest is the request body for POST /api/events/:id/register.
// Email always comes from the authenticated account; it is not accepted in the body.
//
// Note 2: Name is optional because authenticated users already have a display
// name in their profile. If not provided, the service falls back to the
// user's profile name.
type CreateRegistrationRequest struct {
	Name *string `json:"name" validate:"omitempty,min=1,max=200"`
}

// CreateGuestRegistrationRequest is the request body for POST /api/events/:id/register/guest.
//
// Note 3: Email is required here (unlike the authenticated flow) because
// guest users have no account. The email is used for OTP verification and
// as the registration identifier.
type CreateGuestRegistrationRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=200"`
	Email string `json:"email" validate:"required,email"`
}

// RequestGuestOTPRequest is the body for POST /api/events/:id/register/guest/otp.
//
// Note 4: Name is included again in the OTP request because the guest's name
// is stored during registration, not during the later OTP verification step.
// This design defers user creation until OTP is verified, avoiding orphan
// records from abandoned OTP flows.
type RequestGuestOTPRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=200"`
	Email string `json:"email" validate:"required,email"`
}

// RequestGuestOTPResponse is returned after an OTP email is queued.
//
// Note 5: ExpiresInSeconds tells the frontend how long to show a
// countdown/retry UI before the user can request a new OTP.
type RequestGuestOTPResponse struct {
	Status           string `json:"status"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// VerifyGuestOTPRequest is the body for POST /api/events/:id/register/guest/otp/verify.
//
// Note 6: The OTP is validated as exactly 6 numeric characters ("len=6,numeric").
// This constraint is enforced both by the validator and the HMAC-based OTP
// generation logic, ensuring consistency.
type VerifyGuestOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6,numeric"`
}

// VerifyGuestOTPResponse is returned after successful email ownership verification.
type VerifyGuestOTPResponse struct {
	Status           string `json:"status"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// UpdateRegistrationStatusRequest is the request body for PATCH /api/registrations/:id/status.
//
// Note 7: The status is restricted to a fixed set via "oneof". This prevents
// invalid state transitions at the API boundary before business logic runs.
type UpdateRegistrationStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending approved rejected cancelled"`
}

// CheckInRequest is the request body for POST /api/registrations/:id/check-in.
//
// Note 8: The access token is a UUID that was generated at registration time
// and shared with the attendee. Presenting it proves ownership of the
// registration without requiring authentication at the check-in desk.
type CheckInRequest struct {
	AccessToken uuid.UUID `json:"access_token" validate:"required"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

// RegistrationResponse is the JSON representation of a registration row.
//
// Note 9: UserID is only set for authenticated registrations. Guest
// registrations leave it null, and the frontend uses its presence to
// differentiate the two registration flows.
//
// Note 10: AccessToken and TokenExpiresAt are only populated when the
// registration is first created. Subsequent GET requests omit them for
// security — the token is a bearer credential and should not be exposed
// in list responses.
type RegistrationResponse struct {
	ID             uuid.UUID  `json:"id"`
	EventID        uuid.UUID  `json:"event_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	AccessToken    *uuid.UUID `json:"access_token,omitempty"`
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
//
// Note 11: pgtype fields (UserID, TokenExpiresAt, CheckedInAt) are converted
// from their .Valid + .Bytes/.Time representation to idiomatic Go pointers,
// matching the same pattern used in NewEventResponse.
func NewRegistrationResponse(r db.Registration) RegistrationResponse {
	resp := RegistrationResponse{
		ID:        r.ID,
		EventID:   r.EventID,
		Name:      r.Name,
		Email:     r.Email,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
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
