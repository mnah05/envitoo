package helper

import "strings"

// NormalizeEmail lowercases and trims an email for stable uniqueness and
// guest-link matching. Empty input stays empty.
//
// By storing emails in a canonical form ("guest@example.com" regardless of
// "Guest@Example.COM") the application avoids duplicate registrations that
// differ only by casing. This is a common defensive pattern in user-facing
// systems where people type their email address inconsistently.
//
// Note 1: Trimming whitespace first then lowercasing ensures that " A@B " and
// "a@b" produce identical results, which is important for both uniqueness
// constraints and guest-link matching downstream.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
