-- name: IsEventOrganizer :one
SELECT EXISTS (
  SELECT 1 FROM events
  WHERE id = $1 AND organizer_id = $2
)::bool;

-- name: IsVolunteerEventAdmin :one
SELECT EXISTS (
  SELECT 1
  FROM volunteer_applications va
  JOIN volunteer_roles vr ON vr.id = va.role_id
  WHERE va.event_id = $1
    AND va.user_id = $2
    AND va.status = 'approved'
    AND vr.is_admin = true
)::bool;

-- name: ListMyVolunteerStaffEvents :many
SELECT
    e.id AS event_id,
    e.title,
    e.slug,
    e.starts_at,
    e.ends_at,
    e.status AS event_status,
    va.id AS application_id,
    va.updated_at AS approved_at,
    vr.id AS role_id,
    vr.name AS role_name,
    vr.is_admin AS is_admin,
    COUNT(*) OVER()::int4 AS total_count
FROM
    volunteer_applications va
    JOIN volunteer_roles vr ON vr.id = va.role_id
    JOIN events e ON e.id = va.event_id
WHERE
    va.user_id = $1
    AND va.status = 'approved'
    AND vr.is_admin = true
ORDER BY e.starts_at DESC
LIMIT $2
OFFSET
    $3;