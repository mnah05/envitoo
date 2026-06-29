-- name: UpsertVolunteerSettings :one
INSERT INTO volunteer_settings
  (event_id, enabled, description, max_applications, created_at, updated_at)
VALUES
  ($1, $2, $3, $4, now(), now())
ON CONFLICT (event_id)
DO UPDATE SET
  enabled = EXCLUDED.enabled,
  description = EXCLUDED.description,
  max_applications = EXCLUDED.max_applications,
  updated_at = now()
RETURNING *;

-- name: GetVolunteerSettingsByEvent :one
SELECT * FROM volunteer_settings
WHERE event_id = $1;

-- name: CreateVolunteerRole :one
INSERT INTO volunteer_roles
  (event_id, name, is_admin, created_at)
VALUES
  ($1, $2, $3, now())
RETURNING *;

-- name: DeleteVolunteerRole :one
DELETE FROM volunteer_roles
WHERE id = $1 AND event_id = $2
RETURNING *;

-- name: GetVolunteerRoleByID :one
SELECT * FROM volunteer_roles
WHERE id = $1;

-- name: ListVolunteerRolesByEvent :many
SELECT * FROM volunteer_roles
WHERE event_id = $1
ORDER BY name;

-- name: CreateVolunteerApplication :one
INSERT INTO volunteer_applications
  (event_id, user_id, role_id, name, email, phone, description, status, created_at, updated_at)
VALUES
  ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
RETURNING *;

-- name: GetVolunteerApplicationWithRole :one
SELECT
  va.*,
  vr.name AS role_name,
  vr.is_admin AS is_admin
FROM volunteer_applications va
JOIN volunteer_roles vr ON va.role_id = vr.id
WHERE va.id = $1;

-- name: GetVolunteerApplicationByID :one
SELECT
  va.*,
  vr.name AS role_name,
  vr.is_admin AS is_admin
FROM volunteer_applications va
JOIN volunteer_roles vr ON va.role_id = vr.id
WHERE va.id = $1 AND va.user_id = $2;

-- name: ListMyVolunteerApplications :many
SELECT
  va.*,
  vr.name AS role_name,
  vr.is_admin AS is_admin,
  COUNT(*) OVER()::int4 AS total_count
FROM volunteer_applications va
JOIN volunteer_roles vr ON va.role_id = vr.id
WHERE va.user_id = $1
ORDER BY va.created_at DESC
LIMIT $2
OFFSET $3;

-- name: CancelVolunteerApplication :one
UPDATE volunteer_applications
   SET status = 'cancelled', updated_at = now()
 WHERE id = $1
   AND user_id = $2
   AND status = 'pending'
RETURNING *;

-- name: GetVolunteerApplicationByIDAdmin :one
SELECT
  va.*,
  vr.name AS role_name,
  vr.is_admin AS is_admin
FROM volunteer_applications va
JOIN volunteer_roles vr ON va.role_id = vr.id
JOIN events e ON e.id = va.event_id
WHERE va.id = $1
  AND (
    e.organizer_id = $2
    OR EXISTS (
      SELECT 1
      FROM volunteer_applications staff
      JOIN volunteer_roles staff_role ON staff_role.id = staff.role_id
      WHERE staff.event_id = e.id
        AND staff.user_id = $2
        AND staff.status = 'approved'
        AND staff_role.is_admin = true
    )
  );

-- name: ListEventVolunteerApplications :many
SELECT
  va.*,
  vr.name AS role_name,
  vr.is_admin AS is_admin,
  COUNT(*) OVER()::int4 AS total_count
FROM volunteer_applications va
JOIN volunteer_roles vr ON va.role_id = vr.id
JOIN events e ON e.id = va.event_id
WHERE va.event_id = $1
  AND (
    e.organizer_id = $2
    OR EXISTS (
      SELECT 1
      FROM volunteer_applications staff
      JOIN volunteer_roles staff_role ON staff_role.id = staff.role_id
      WHERE staff.event_id = e.id
        AND staff.user_id = $2
        AND staff.status = 'approved'
        AND staff_role.is_admin = true
    )
  )
ORDER BY va.created_at DESC
LIMIT $3
OFFSET $4;

-- name: CountVolunteerApplicationsByEvent :one
SELECT COUNT(*)::int4
FROM volunteer_applications
WHERE event_id = $1
  AND status IN ('pending', 'approved');

-- name: ApproveVolunteerApplication :one
-- Volunteer admins may only approve applications for non-admin roles;
-- admin-role approvals are reserved for the event organizer.
UPDATE volunteer_applications va
   SET status = 'approved', updated_at = now()
  FROM events e, volunteer_roles target
 WHERE va.id = $1
   AND e.id = va.event_id
   AND target.id = va.role_id
   AND (
     e.organizer_id = $2
     OR (
       target.is_admin = false
       AND EXISTS (
         SELECT 1
         FROM volunteer_applications staff
         JOIN volunteer_roles staff_role ON staff_role.id = staff.role_id
         WHERE staff.event_id = e.id
           AND staff.user_id = $2
           AND staff.status = 'approved'
           AND staff_role.is_admin = true
       )
     )
   )
    AND va.status = 'pending'
RETURNING va.*;

-- name: ListEventVolunteers :many
-- Returns confirmed volunteers for an event. Authorization is enforced
-- in SQL: the organizer or an approved volunteer admin may read.
SELECT
  v.id, v.event_id, v.user_id, v.role, v.created_at,
  u.email, u.username, u.avatar_url,
  COUNT(*) OVER()::int4 AS total_count
FROM volunteers v
JOIN users u ON u.id = v.user_id
JOIN events e ON e.id = v.event_id
WHERE v.event_id = $1
  AND (
    e.organizer_id = $2
    OR EXISTS (
      SELECT 1
      FROM volunteer_applications staff
      JOIN volunteer_roles staff_role ON staff_role.id = staff.role_id
      WHERE staff.event_id = e.id
        AND staff.user_id = $2
        AND staff.status = 'approved'
        AND staff_role.is_admin = true
    )
  )
ORDER BY v.created_at DESC
LIMIT $3
OFFSET $4;

-- name: InsertVolunteer :one
-- Records a confirmed volunteer in the volunteers roster table.
-- ON CONFLICT DO NOTHING makes this idempotent — if the volunteer
-- row already exists (e.g. re-approval), no row is returned and the
-- caller treats pgx.ErrNoRows as success.
INSERT INTO volunteers (event_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (event_id, user_id, role) DO NOTHING
RETURNING *;

-- name: RejectVolunteerApplication :one
-- Volunteer admins may only reject applications for non-admin roles;
-- admin-role rejections are reserved for the event organizer.
UPDATE volunteer_applications va
   SET status = 'rejected', updated_at = now()
  FROM events e, volunteer_roles target
 WHERE va.id = $1
   AND e.id = va.event_id
   AND target.id = va.role_id
   AND (
     e.organizer_id = $2
     OR (
       target.is_admin = false
       AND EXISTS (
         SELECT 1
         FROM volunteer_applications staff
         JOIN volunteer_roles staff_role ON staff_role.id = staff.role_id
         WHERE staff.event_id = e.id
           AND staff.user_id = $2
           AND staff.status = 'approved'
           AND staff_role.is_admin = true
       )
     )
   )
   AND va.status = 'pending'
RETURNING va.*;