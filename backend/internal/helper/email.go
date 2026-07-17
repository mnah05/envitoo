package helper

import "strings"

// NormalizeEmail lowercases and trims an email for stable uniqueness and
// guest-link matching. Empty input stays empty.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
