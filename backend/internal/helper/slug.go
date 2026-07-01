package helper

import (
	"crypto/rand"
)

const eventSlugLength = 7

// slugAlphabet is lowercase ASCII letters and digits — URL-safe, no hyphens.
const slugAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateEventSlug returns a random 7-character URL-safe slug (a-z, 0-9).
func GenerateEventSlug() (string, error) {
	return randomSlug(eventSlugLength)
}

func randomSlug(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range b {
		out[i] = slugAlphabet[int(b[i])%len(slugAlphabet)]
	}
	return string(out), nil
}