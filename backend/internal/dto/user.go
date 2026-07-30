package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

// UserResponse is the JSON representation of the authenticated local user.
// sqlc models have no json tags, so handlers must not serialize db.User directly.
//
// Note 1: AuthUserID is the external user ID from Clerk (the authentication
// provider). This is distinct from our internal UUID primary key (ID) and is
// used to look up the user during Clerk webhook processing.
//
// Note 2: Username and AvatarURL are optional — they depend on the user's
// Clerk profile settings. The omitempty tag excludes them when nil, keeping
// the response compact for minimal profiles.
type UserResponse struct {
	ID         uuid.UUID `json:"id"`
	AuthUserID string    `json:"auth_user_id"`
	Email      string    `json:"email"`
	Username   *string   `json:"username,omitempty"`
	AvatarURL  *string   `json:"avatar_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewUserResponse maps a db.User row to the public API shape.
//
// Note 3: Accepting a pointer (*db.User) allows the caller to pass nil for
// "user not found" without making the response construction awkward. The
// returned zero-value UserResponse has empty UUIDs and strings, which
// serializes to {"id":"00000000-..."} — typically the caller handles nil
// differently, but the DTO stays defensive.
func NewUserResponse(u *db.User) UserResponse {
	if u == nil {
		return UserResponse{}
	}
	return UserResponse{
		ID:         u.ID,
		AuthUserID: u.AuthUserID,
		Email:      u.Email,
		Username:   u.Username,
		AvatarURL:  u.AvatarUrl,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}
