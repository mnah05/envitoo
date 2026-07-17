package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/mnah05/envitoo/internal/db"
)

// UserResponse is the JSON representation of the authenticated local user.
// sqlc models have no json tags, so handlers must not serialize db.User directly.
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
