-- name: UpdateEventStaffFields :one
UPDATE events e
SET
    description = $2,
    cover_image_url = $3,
    meeting_url = $4,
    venue_name = $5,
    location_address = $6,
    lat = $7,
    lng = $8,
    updated_at = now()
WHERE
    e.id = $1
    AND (
        e.organizer_id = $9
        OR EXISTS (
            SELECT 1
            FROM
                volunteer_applications va
                JOIN volunteer_roles vr ON vr.id = va.role_id
            WHERE
                va.event_id = e.id
                AND va.user_id = $8
                AND va.status = 'approved'
                AND vr.is_admin = true
        )
    ) RETURNING *;