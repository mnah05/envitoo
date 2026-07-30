package helper

import (
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// Note 1: These tests wrap pgconn.PgError with fmt.Errorf to simulate how
// database errors actually appear in production — wrapped with context.
// Testing with the unwrapped error alone would miss error-chain unwrapping
// issues in the implementation.

func TestIsUniqueViolation_pgxv5(t *testing.T) {
	err := &pgconn.PgError{
		Code:           pgerrcode.UniqueViolation,
		ConstraintName: "idx_registrations_event_email_active",
	}
	wrapped := fmt.Errorf("create registration: %w", err)

	// Match by exact constraint name
	if !IsUniqueViolation(wrapped, "idx_registrations_event_email_active") {
		t.Fatal("expected unique violation match for pgx/v5/pgconn.PgError")
	}
	// The same error should also match IsRegistrationDuplicate
	if !IsRegistrationDuplicate(wrapped) {
		t.Fatal("expected IsRegistrationDuplicate for idx_registrations_event_email_active")
	}
	// A different constraint name should NOT match — confirms the constraint
	// filtering works correctly
	if IsUniqueViolation(wrapped, "other_constraint") {
		t.Fatal("did not expect match for different constraint")
	}
}

func TestIsRegistrationDuplicate_constraintNames(t *testing.T) {
	// Iterate over all three constraint names that IsRegistrationDuplicate
	// should accept, including a pre-migration fallback name.
	names := []string{
		"idx_registrations_event_email_active",
		"idx_registrations_user_event",
		"registrations_event_id_email_key", // pre-migration
	}
	for _, name := range names {
		err := fmt.Errorf("create: %w", &pgconn.PgError{
			Code:           pgerrcode.UniqueViolation,
			ConstraintName: name,
		})
		if !IsRegistrationDuplicate(err) {
			t.Fatalf("IsRegistrationDuplicate(%q) = false, want true", name)
		}
	}
	// Confirm that an unrelated constraint name is rejected
	if IsRegistrationDuplicate(fmt.Errorf("create: %w", &pgconn.PgError{
		Code:           pgerrcode.UniqueViolation,
		ConstraintName: "other_constraint",
	})) {
		t.Fatal("did not expect match for unrelated constraint")
	}
}

func TestIsVolunteerApplicationDuplicate_pgxv5(t *testing.T) {
	err := &pgconn.PgError{
		Code:           pgerrcode.UniqueViolation,
		ConstraintName: "idx_volunteer_apps_event_user_active",
	}
	if !IsVolunteerApplicationDuplicate(fmt.Errorf("apply: %w", err)) {
		t.Fatal("expected volunteer application duplicate match")
	}
}

func TestIsForeignKeyViolation_pgxv5(t *testing.T) {
	err := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}
	if !IsForeignKeyViolation(fmt.Errorf("delete: %w", err)) {
		t.Fatal("expected foreign key violation match")
	}
}
