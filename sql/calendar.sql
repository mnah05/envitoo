-- name: GetCalendarByUserID :one
SELECT * FROM calendars WHERE user_id = $1 LIMIT 1;

-- name: CreateCalendar :one
INSERT INTO calendars (user_id, slug, name)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
RETURNING *;
