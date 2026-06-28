package helper

import (
	"github.com/google/uuid"
)

// GenerateAccessToken returns a new random UUID v4 suitable for use as
// a registration access token. The registrations table enforces
// uniqueness via uq_access_token. In the astronomically unlikely event
// of a collision the caller should retry with a newly generated token.
func GenerateAccessToken() uuid.UUID {
	return uuid.Must(uuid.NewRandom())
}
