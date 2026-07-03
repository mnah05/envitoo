-- name: UpsertEventVisibility :one
INSERT INTO event_visibility (event_id, visibility)
VALUES ($1, $2)
ON CONFLICT (event_id) DO UPDATE SET visibility = $2
RETURNING *;

-- name: GetEventVisibility :one
SELECT * FROM event_visibility WHERE event_id = $1;

-- name: ListEventVisibilityByIDs :many
SELECT * FROM event_visibility WHERE event_id = ANY($1::uuid[]);

-- name: DeleteEventVisibility :exec
DELETE FROM event_visibility WHERE event_id = $1;