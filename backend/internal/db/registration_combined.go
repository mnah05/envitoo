package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const checkOrphanedGuestRegistrations = `SELECT EXISTS (SELECT 1 FROM registrations WHERE email = $1 AND user_id IS NULL)`

func (q *Queries) CheckOrphanedGuestRegistrations(ctx context.Context, email string) (bool, error) {
	row := q.db.QueryRow(ctx, checkOrphanedGuestRegistrations, email)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

// The linked and cancelled CTEs below are disjoint: a given row matches at
// most one of them. Link targets rows where NO user registration exists for
// that event (NOT EXISTS subquery). Cancel targets rows where a user
// registration DOES exist (EXISTS subquery). The complementary EXISTS vs
// NOT EXISTS on the identical subquery guarantees no row can satisfy both
// conditions, so the two sets can be counted independently in a single pass.
//
// This replaces the separate LinkGuestRegistrations and
// CancelSupersededGuestRegistrations sqlc queries with a single round-trip.
const linkGuestRegistrationsCombined = `WITH linked AS (
	UPDATE registrations g
	   SET user_id = $2, updated_at = now()
	 WHERE g.email = $1
	   AND g.user_id IS NULL
	   AND NOT EXISTS (
		 SELECT 1 FROM registrations u
		 WHERE u.event_id = g.event_id
		   AND u.user_id = $2
		   AND u.status IN ('pending', 'approved')
	   )
	 RETURNING 1
), cancelled AS (
	UPDATE registrations g
	   SET status = 'cancelled', updated_at = now()
	 WHERE g.email = $1
	   AND g.user_id IS NULL
	   AND g.status NOT IN ('cancelled', 'rejected')
	   AND EXISTS (
		 SELECT 1 FROM registrations u
		 WHERE u.event_id = g.event_id
		   AND u.user_id = $2
		   AND u.status IN ('pending', 'approved')
	   )
	 RETURNING 1
)
SELECT (SELECT count(*) FROM linked) AS linked_count,
	   (SELECT count(*) FROM cancelled) AS cancelled_count`

func (q *Queries) LinkGuestRegistrationsCombined(ctx context.Context, email string, userID pgtype.UUID) (linked, cancelled int64, err error) {
	row := q.db.QueryRow(ctx, linkGuestRegistrationsCombined, email, userID)
	err = row.Scan(&linked, &cancelled)
	return
}
