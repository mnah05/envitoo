package helper

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	multipleDashes  = regexp.MustCompile(`-{2,}`)
)

// Generate creates a URL-safe slug from a title with a random suffix.
// e.g. "Tech Fest 2025!" -> "tech-fest-2025-a3f9c1b2"
func Generate(title string) (string, error) {
	base := toBase(title)
	if base == "" {
		base = "event"
	}

	suffix, err := randomHex(4) // 4 bytes = 8 hex chars
	if err != nil {
		return "", err
	}

	return base + "-" + suffix, nil
}

func toBase(title string) string {
	// Normalize unicode: "café" -> "cafe"
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r) // strip combining marks
	}), norm.NFC)
	normalized, _, _ := transform.String(t, title)

	lower := strings.ToLower(normalized)
	dashed := nonAlphanumeric.ReplaceAllString(lower, "-")
	clean := multipleDashes.ReplaceAllString(dashed, "-")
	clean = strings.Trim(clean, "-")

	// Cap base at 50 chars, don't cut mid-word
	if len(clean) > 50 {
		clean = clean[:50]
		if idx := strings.LastIndex(clean, "-"); idx > 0 {
			clean = clean[:idx]
		}
	}

	return clean
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
