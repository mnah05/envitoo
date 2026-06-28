package helper

import (
	"errors"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
)

// IsUniqueViolation returns true if err is a Postgres unique violation
// on the named constraint. Use it to detect collisions and retry.
//
// Note: When using ON CONFLICT DO NOTHING with RETURNING *, a collision
// produces pgx.ErrNoRows rather than a unique violation error. In that
// case check for pgx.ErrNoRows directly.
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
// the registrations (event_id, email) or (event_id, user_id) constraints.
func IsRegistrationDuplicate(err error) bool {
	return IsUniqueViolation(err, "registrations_event_id_email_key") ||
		IsUniqueViolation(err, "idx_registrations_user_event")
}
