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
//
// Note 1: The function accepts any error type and uses errors.As to unwrap
// wrapped errors. This is important because callers typically wrap database
// errors with context like fmt.Errorf("create registration: %w", err).
// errors.As walks the error chain to find a matching *pgconn.PgError.
//
// Note 2: Checking both the error code (UniqueViolation) AND the constraint
// name prevents false positives. A unique violation on a different table
// would have a different constraint name, so the caller gets a precise match.
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
//
// Note 3: Partial unique indexes are used instead of a full unique constraint
// because the application uses soft-delete (active = false) instead of hard
// DELETE. A regular UNIQUE constraint would prevent re-registration even
// after cancellation. The partial index only enforces uniqueness where
// active = true.
//
// Note 4: Three constraint names are checked because migration renames are
// not atomic across all environments. The old name
// "registrations_event_id_email_key" is checked as a fallback for databases
// that haven't been migrated yet.
func IsRegistrationDuplicate(err error) bool {
	return IsUniqueViolation(err, "idx_registrations_event_email_active") ||
		IsUniqueViolation(err, "idx_registrations_user_event") ||
		IsUniqueViolation(err, "registrations_event_id_email_key")
}

// IsForeignKeyViolation returns true when err is a Postgres foreign-key
// violation, e.g. ON DELETE RESTRICT blocking a parent delete.
//
// Note 5: Unlike IsUniqueViolation, this function does not check a specific
// constraint name because foreign key violations are typically surfaced as
// generic errors to the API caller (e.g. "cannot delete, event has
// registrations"). The exact constraint name is not useful information at
// that level.
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
//
// Note 6: Like registrations, volunteer applications use soft-delete and a
// partial unique index to prevent duplicate active applications while
// preserving history of previous ones.
func IsVolunteerApplicationDuplicate(err error) bool {
	return IsUniqueViolation(err, "idx_volunteer_apps_event_user_active")
}
