-- name: CreateEvent :one
INSERT INTO
    events (
        organizer_id,
        title,
        description,
        starts_at,
        ends_at,
        capacity,
        price_in_paise,
        currency,
        status,
        requires_approval,
        cover_image_url,
        calendar_id,
        is_virtual,
        meeting_url,
        venue_name,
        location_address,
        lat,
        lng,
        slug,
        category
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11,
        $12,
        $13,
        $14,
        $15,
        $16,
        $17,
        $18,
        $19,
        $20
    ) ON CONFLICT (slug) DO NOTHING
RETURNING *;

-- name: GetEventByID :one
SELECT * FROM events WHERE id = $1;

-- name: GetEventByIDForUpdate :one
SELECT * FROM events WHERE id = $1 FOR UPDATE;

-- name: GetEventBySlug :one
SELECT * FROM events WHERE slug = $1;

-- name: GetPublishedEventByID :one
SELECT * FROM events WHERE id = $1 AND status = 'published';

-- name: GetPublishedEventBySlug :one
SELECT * FROM events WHERE slug = $1 AND status = 'published';

-- name: CountEventsByOrganizer :one
SELECT COUNT(*)::int4
FROM events
WHERE
    organizer_id = $1
    AND ($2::boolean = true OR status != 'cancelled');

-- name: ListPublicEvents :many
SELECT *
FROM events
WHERE
    status = 'published'
    AND ($1::text = '' OR category = $1)
    AND ($2::int = 0 OR ($2::int = 1 AND is_virtual = true) OR ($2::int = 2 AND is_virtual = false))
    AND ($3::text = '' OR title ILIKE '%' || $3 || '%' OR description ILIKE '%' || $3 || '%')
    AND (starts_at >= $4 OR $4 = '1970-01-01T00:00:00Z'::timestamptz)
    AND (starts_at <= $5 OR $5 = '9999-12-31T23:59:59Z'::timestamptz)
ORDER BY starts_at ASC
LIMIT $6 OFFSET $7;

-- name: CountPublicEvents :one
SELECT COUNT(*)::int4
FROM events
WHERE
    status = 'published'
    AND ($1::text = '' OR category = $1)
    AND ($2::int = 0 OR ($2::int = 1 AND is_virtual = true) OR ($2::int = 2 AND is_virtual = false))
    AND ($3::text = '' OR title ILIKE '%' || $3 || '%' OR description ILIKE '%' || $3 || '%')
    AND (starts_at >= $4 OR $4 = '1970-01-01T00:00:00Z'::timestamptz)
    AND (starts_at <= $5 OR $5 = '9999-12-31T23:59:59Z'::timestamptz);

-- name: ListEventsByOrganizer :many
SELECT *
FROM events
WHERE
    organizer_id = $1
    AND (@include_cancelled::boolean = true OR status != 'cancelled')
ORDER BY starts_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateEvent :one
UPDATE events
SET
    description = $2,
    starts_at = $3,
    ends_at = $4,
    capacity = $5,
    price_in_paise = $6,
    currency = $7,
    requires_approval = $8,
    cover_image_url = $9,
    calendar_id = $10,
    is_virtual = $11,
    meeting_url = $12,
    venue_name = $13,
    location_address = $14,
    lat = $15,
    lng = $16,
    category = $17,
    updated_at = now()
WHERE
    id = $1
    AND organizer_id = $18
RETURNING *;

-- name: CancelEvent :one
UPDATE events
SET
    status = 'cancelled',
    updated_at = now()
WHERE
    id = $1
    AND organizer_id = $2 RETURNING *;