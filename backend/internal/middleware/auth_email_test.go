package middleware

import (
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
)

func TestPrimaryEmail_requiresVerified(t *testing.T) {
	// Test that primaryEmail prefers the primary address only when it is
	// verified, and falls back to the first verified address otherwise.
	// If no email is verified, it returns "" which triggers a 403 in the
	// LoadUser middleware.
	unverified := &clerk.EmailAddress{
		ID:           "idn_unverified",
		EmailAddress: "raw@example.com",
		Verification: &clerk.Verification{Status: "unverified"},
	}
	verified := &clerk.EmailAddress{
		ID:           "idn_verified",
		EmailAddress: "ok@example.com",
		Verification: &clerk.Verification{Status: "verified"},
	}
	primaryID := "idn_unverified"

	u := &clerk.User{
		PrimaryEmailAddressID: &primaryID,
		EmailAddresses:        []*clerk.EmailAddress{unverified, verified},
	}

	// Primary is unverified → fall through to first verified.
	if got := primaryEmail(u); got != "ok@example.com" {
		t.Fatalf("primaryEmail() = %q, want verified address", got)
	}

	// Only unverified → empty (403 path).
	u.EmailAddresses = []*clerk.EmailAddress{unverified}
	if got := primaryEmail(u); got != "" {
		t.Fatalf("primaryEmail() = %q, want empty when none verified", got)
	}

	// Verified primary preferred.
	primaryID = "idn_verified"
	u.PrimaryEmailAddressID = &primaryID
	u.EmailAddresses = []*clerk.EmailAddress{verified, {
		ID:           "idn_other",
		EmailAddress: "other@example.com",
		Verification: &clerk.Verification{Status: "verified"},
	}}
	if got := primaryEmail(u); got != "ok@example.com" {
		t.Fatalf("primaryEmail() = %q, want primary verified", got)
	}
}
