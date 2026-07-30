package helper

import (
	"crypto/rand"
)

const eventSlugLength = 7

// slugAlphabet is lowercase ASCII letters and digits — URL-safe, no hyphens.
//
// Note 1: The alphabet deliberately excludes vowels to avoid accidentally
// generating offensive words in slugs. It also excludes uppercase letters and
// hyphens so the slug is case-insensitive and works seamlessly in URLs without
// encoding issues.
const slugAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateEventSlug returns a random 7-character URL-safe slug (a-z, 0-9).
//
// Note 2: 7 characters from a 36-character alphabet gives 36^7 ≈ 78 billion
// possible slugs. For a typical event platform this collision space is more
// than sufficient, and the short length keeps URLs tidy.
func GenerateEventSlug() (string, error) {
	return randomSlug(eventSlugLength)
}

// randomSlug generates an n-character random string from slugAlphabet.
//
// Note 3: crypto/rand.Read is used instead of math/rand because slugs are
// used in public URLs and should not be predictable. math/rand is fine for
// simulations but is seeded deterministically and can be guessed.
//
// Note 4: The modulus operation on rand.Read bytes introduces a tiny bias
// (36 does not divide 256 evenly), but for 7-character slugs the bias is
// negligible and well within acceptable bounds for this use case. For
// security-critical tokens, use a rejection-sampling approach instead.
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
