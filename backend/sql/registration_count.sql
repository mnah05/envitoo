-- name: GetRegistrationCountsByEventIDs :many
SELECT event_id, COUNT(*)::int4 AS count
FROM registrations
WHERE event_id = ANY($1::uuid[])
  AND status NOT IN ('cancelled', 'rejected')
GROUP BY event_id;
