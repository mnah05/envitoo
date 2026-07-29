package helper

import (
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation_pgxv5(t *testing.T) {
	err := &pgconn.PgError{
		Code:           pgerrcode.UniqueViolation,
		ConstraintName: "idx_registrations_event_email_active",
	}
	wrapped := fmt.Errorf("create registration: %w", err)

	if !IsUniqueViolation(wrapped, "idx_registrations_event_email_active") {
		t.Fatal("expected unique violation match for pgx/v5/pgconn.PgError")
	}
	if !IsRegistrationDuplicate(wrapped) {
		t.Fatal("expected IsRegistrationDuplicate for idx_registrations_event_email_active")
	}
	if IsUniqueViolation(wrapped, "other_constraint") {
		t.Fatal("did not expect match for different constraint")
	}
}

func TestIsRegistrationDuplicate_constraintNames(t *testing.T) {
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
