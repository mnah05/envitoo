package helper

import (
	"github.com/google/uuid"
)

// GenerateAccessToken returns a new random UUID v4 suitable for use as
// a registration access token. The registrations table enforces
// uniqueness via uq_access_token. In the astronomically unlikely event
// of a collision the caller should retry with a newly generated token.
//
// Note 1: uuid.NewRandom() generates a UUID version 4 based on
// cryptographic randomness, producing 122 random bits. The odds of collision
// across 103 trillion version-4 UUIDs is about one in a billion, making it
// safe for uncoordinated generation across distributed workers.
//
// Note 2: uuid.Must() panics if NewRandom() returns an error (which only
// happens if the system's random source fails). Panicking here is a reasonable
// choice because a failed random read signals a catastrophic system issue that
// would prevent any further useful work.
func GenerateAccessToken() uuid.UUID {
	return uuid.Must(uuid.NewRandom())
}
