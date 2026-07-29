package helper

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation returns true if err is a Postgres unique violation
// on the named constraint. Use it to detect collisions and retry.
//
// Note: When using ON CONFLICT DO NOTHING with RETURNING *, a collision
// produces pgx.ErrNoRows rather than a unique violation error. In that
// case check for pgx.ErrNoRows directly.
//
// Must use github.com/jackc/pgx/v5/pgconn.PgError — the legacy
// github.com/jackc/pgconn.PgError type is a different type, so
// errors.As never matches errors returned by pgx/v5.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if err == nil {
		return false
	}
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == constraint
	}
	return false
}

// IsRegistrationDuplicate returns true when err is a unique violation on
// the active-registration partial unique indexes (event_id, email) or
// (event_id, user_id). Also matches the pre-migration table constraint
// name so unmigrated databases still map to ErrConflict.
func IsRegistrationDuplicate(err error) bool {
	return IsUniqueViolation(err, "idx_registrations_event_email_active") ||
		IsUniqueViolation(err, "idx_registrations_user_event") ||
		IsUniqueViolation(err, "registrations_event_id_email_key")
}

// IsForeignKeyViolation returns true when err is a Postgres foreign-key
// violation, e.g. ON DELETE RESTRICT blocking a parent delete.
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if err == nil {
		return false
	}
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.ForeignKeyViolation
	}
	return false
}

// IsVolunteerApplicationDuplicate returns true when err is a unique violation
// on the active volunteer application (event_id, user_id) partial index.
func IsVolunteerApplicationDuplicate(err error) bool {
	return IsUniqueViolation(err, "idx_volunteer_apps_event_user_active")
}
