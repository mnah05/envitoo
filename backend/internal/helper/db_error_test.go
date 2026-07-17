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
		ConstraintName: "registrations_event_id_email_key",
	}
	wrapped := fmt.Errorf("create registration: %w", err)

	if !IsUniqueViolation(wrapped, "registrations_event_id_email_key") {
		t.Fatal("expected unique violation match for pgx/v5/pgconn.PgError")
	}
	if !IsRegistrationDuplicate(wrapped) {
		t.Fatal("expected IsRegistrationDuplicate for registrations_event_id_email_key")
	}
	if IsUniqueViolation(wrapped, "other_constraint") {
		t.Fatal("did not expect match for different constraint")
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
