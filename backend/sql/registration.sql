-- name: CountActiveRegistrationsByEvent :one
SELECT COUNT(*)::int4
FROM registrations
WHERE event_id = $1
  AND status NOT IN ('cancelled', 'rejected');

-- name: LinkGuestRegistrations :execrows
UPDATE registrations g
   SET user_id = $2, updated_at = now()
 WHERE g.email = $1
   AND g.user_id IS NULL
   AND NOT EXISTS (
     SELECT 1 FROM registrations u
     WHERE u.event_id = g.event_id
       AND u.user_id = $2
   );

-- name: CancelSupersededGuestRegistrations :execrows
UPDATE registrations g
   SET status = 'cancelled', updated_at = now()
 WHERE g.email = $1
   AND g.user_id IS NULL
   AND g.status NOT IN ('cancelled', 'rejected')
   AND EXISTS (
     SELECT 1 FROM registrations u
     WHERE u.event_id = g.event_id
       AND u.user_id = $2
   );

-- Route 1: POST /api/events/:id/register/guest
-- name: CreateRegistrationGuest :one
INSERT INTO registrations
  (event_id, user_id, name, email, status, created_at, updated_at, access_token, token_expires_at, checked_in_at)
VALUES
  ($1, NULL, $2, $3, $4, now(), now(), $5, $6, NULL)
ON CONFLICT ON CONSTRAINT uq_access_token DO NOTHING
RETURNING *;

-- Route 2: GET /api/guest-registrations?token=
-- name: GetRegistrationByAccessToken :one
SELECT r.*
FROM registrations r
WHERE r.access_token = $1
  AND (r.token_expires_at IS NULL OR r.token_expires_at > now());

-- Route 3: DELETE /api/guest-registrations?token=
-- name: CancelGuestRegistration :one
UPDATE registrations
   SET status = 'cancelled', updated_at = now()
 WHERE access_token = $1
   AND user_id IS NULL
RETURNING *;

-- Route 4: POST /api/events/:id/register
-- name: CreateRegistrationUser :one
INSERT INTO registrations
  (event_id, user_id, name, email, status, created_at, updated_at, access_token, token_expires_at, checked_in_at)
VALUES
  ($1, $2, $3, $4, $5, now(), now(), $6, $7, NULL)
ON CONFLICT ON CONSTRAINT uq_access_token DO NOTHING
RETURNING *;

-- Route 5: GET /api/registrations/:id
-- name: GetRegistrationByID :one
SELECT r.*
FROM registrations r
WHERE r.id = $1
  AND r.user_id = $2;

-- Route 6: DELETE /api/registrations/:id
-- name: CancelRegistration :one
UPDATE registrations
   SET status = 'cancelled', updated_at = now()
 WHERE id = $1
   AND user_id = $2
RETURNING *;

-- Route 7: GET /api/me/registrations
-- name: ListMyRegistrations :many
SELECT r.*, COUNT(*) OVER()::int4 AS total_count
FROM registrations r
WHERE r.user_id = $1
ORDER BY r.created_at DESC
LIMIT $2
OFFSET $3;

-- Route 8: GET /api/events/:id/registrations
-- name: ListEventRegistrations :many
SELECT r.*, COUNT(*) OVER()::int4 AS total_count
FROM registrations r
JOIN events e ON e.id = r.event_id
WHERE r.event_id = $1
  AND (
    e.organizer_id = $2
    OR EXISTS (
      SELECT 1
      FROM volunteer_applications va
      JOIN volunteer_roles vr ON vr.id = va.role_id
      WHERE va.event_id = e.id
        AND va.user_id = $2
        AND va.status = 'approved'
        AND vr.is_admin = true
    )
  )
ORDER BY r.created_at DESC
LIMIT $3
OFFSET $4;

-- Route 9: GET /api/registrations/:id/admin
-- name: GetRegistrationByIDAdmin :one
SELECT r.*
FROM registrations r
JOIN events e ON e.id = r.event_id
WHERE r.id = $1
  AND (
    e.organizer_id = $2
    OR EXISTS (
      SELECT 1
      FROM volunteer_applications va
      JOIN volunteer_roles vr ON vr.id = va.role_id
      WHERE va.event_id = e.id
        AND va.user_id = $2
        AND va.status = 'approved'
        AND vr.is_admin = true
    )
  );

-- Route 10: POST /api/registrations/:id/approve
-- name: ApproveRegistration :one
UPDATE registrations r
   SET status = 'approved', updated_at = now()
  FROM events e
 WHERE r.id = $1
   AND e.id = r.event_id
   AND (
     e.organizer_id = $2
     OR (
       e.requires_approval = true
       AND EXISTS (
         SELECT 1
         FROM volunteer_applications va
         JOIN volunteer_roles vr ON vr.id = va.role_id
         WHERE va.event_id = e.id
           AND va.user_id = $2
           AND va.status = 'approved'
           AND vr.is_admin = true
       )
     )
   )
   AND r.status IN ('pending', 'rejected', 'cancelled')
RETURNING r.*;

-- Route 11: POST /api/registrations/:id/reject
-- name: RejectRegistration :one
UPDATE registrations r
   SET status = 'rejected', updated_at = now()
  FROM events e
 WHERE r.id = $1
   AND e.id = r.event_id
   AND (
     e.organizer_id = $2
     OR (
       e.requires_approval = true
       AND EXISTS (
         SELECT 1
         FROM volunteer_applications va
         JOIN volunteer_roles vr ON vr.id = va.role_id
         WHERE va.event_id = e.id
           AND va.user_id = $2
           AND va.status = 'approved'
           AND vr.is_admin = true
       )
     )
   )
   AND r.status = 'pending'
RETURNING r.*;

-- Route 12: PATCH /api/registrations/:id/status
-- name: UpdateRegistrationStatus :one
UPDATE registrations r
   SET status = $3, updated_at = now()
  FROM events e
 WHERE r.id = $1
   AND e.id = r.event_id
   AND e.organizer_id = $2
RETURNING r.*;

-- Route 13: POST /api/registrations/:id/check-in
-- name: CheckInRegistration :one
UPDATE registrations r
   SET checked_in_at = now(), updated_at = now()
  FROM events e
 WHERE r.id = $1
   AND e.id = r.event_id
   AND (
     e.organizer_id = $2
     OR EXISTS (
       SELECT 1
       FROM volunteer_applications va
       JOIN volunteer_roles vr ON vr.id = va.role_id
       WHERE va.event_id = e.id
         AND va.user_id = $2
         AND va.status = 'approved'
         AND vr.is_admin = true
     )
   )
   AND e.status = 'published'
   AND r.access_token = $3
   AND r.status = 'approved'
   AND r.checked_in_at IS NULL
RETURNING r.*;

-- Route 14: POST /api/events/:id/check-in
-- name: CheckInRegistrationByEventToken :one
UPDATE registrations r
   SET checked_in_at = now(), updated_at = now()
  FROM events e
 WHERE r.event_id = $1
   AND e.id = r.event_id
   AND (
     e.organizer_id = $2
     OR EXISTS (
       SELECT 1
       FROM volunteer_applications va
       JOIN volunteer_roles vr ON vr.id = va.role_id
       WHERE va.event_id = e.id
         AND va.user_id = $2
         AND va.status = 'approved'
         AND vr.is_admin = true
     )
   )
   AND e.status = 'published'
   AND r.access_token = $3
   AND r.status = 'approved'
   AND r.checked_in_at IS NULL
RETURNING r.*;

-- name: GetRegistrationByEventTokenAdmin :one
SELECT r.*
FROM registrations r
JOIN events e ON e.id = r.event_id
WHERE r.event_id = $1
  AND r.access_token = $2
  AND (
    e.organizer_id = $3
    OR EXISTS (
      SELECT 1
      FROM volunteer_applications va
      JOIN volunteer_roles vr ON vr.id = va.role_id
      WHERE va.event_id = e.id
        AND va.user_id = $3
        AND va.status = 'approved'
        AND vr.is_admin = true
    )
  );
