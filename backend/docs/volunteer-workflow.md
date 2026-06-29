# Volunteer Workflow — Implementation Plan

## Overview

Volunteers can apply to events with a specific role + work description. Hosts review applications and accept/reject. On acceptance, the volunteer is confirmed. Hosts can configure roles per event and mark roles as `is_admin` to grant special permissions.

## Auth Model

- **Public endpoints** — no auth required, anyone can read event volunteer settings and roles.
- **Volunteer endpoints** — require Clerk authentication (signed-in users).
- **Organizer endpoints** — require authentication + must be the event owner. Checked in service layer.

## Database Schema

### `volunteer_settings` (1:1 with events)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| id | UUID | PK, gen_random_uuid | |
| event_id | UUID | FK → events(id) ON DELETE CASCADE, UNIQUE | One per event |
| enabled | BOOLEAN | NOT NULL DEFAULT false | Host opens/closes |
| description | TEXT | nullable | Host's CTA text |
| max_applications | INT4 | nullable, min=1 | null = unlimited |
| created_at | timestamptz | NOT NULL DEFAULT now() | |
| updated_at | timestamptz | NOT NULL DEFAULT now() | |

### `volunteer_roles` (host-defined per event)

| Column | Type | Constraints | Notes |
|---|---|---|---|
| id | UUID | PK, gen_random_uuid | |
| event_id | UUID | FK → events(id) ON DELETE CASCADE | |
| name | TEXT | NOT NULL | e.g. "Stage Manager" |
| is_admin | BOOLEAN | NOT NULL DEFAULT false | Grants admin perms |
| created_at | timestamptz | NOT NULL DEFAULT now() | |
| UNIQUE | (event_id, name) | No duplicate names per event |
| INDEX | idx_volunteer_roles_event_id | |

### `volunteer_applications`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| id | UUID | PK, gen_random_uuid | |
| event_id | UUID | FK → events(id) ON DELETE CASCADE | |
| user_id | UUID | FK → users(id) ON DELETE CASCADE | Signed-in user |
| role_id | UUID | FK → volunteer_roles(id) ON DELETE RESTRICT | Selected from dropdown |
| name | TEXT | NOT NULL | From user profile or override |
| email | TEXT | NOT NULL | From user profile or override |
| phone | TEXT | nullable | |
| description | TEXT | NOT NULL | Work background / skills |
| status | TEXT | NOT NULL DEFAULT 'pending' | CHECK ('pending','approved','rejected') |
| created_at | timestamptz | NOT NULL DEFAULT now() | |
| updated_at | timestamptz | NOT NULL DEFAULT now() | |
| UNIQUE | (event_id, user_id) | One per user per event |
| INDEX | idx_volunteer_apps_event_id | |
| INDEX | idx_volunteer_apps_user_id | |
| INDEX | idx_volunteer_apps_status | |

## API Endpoints

### Public (no auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/events/:id/volunteer/settings | Get settings (enabled, description, max) |
| GET | /api/events/:id/volunteer/roles | List available roles |

### Volunteer (auth user)

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/events/:id/volunteer/apply | Submit application |
| GET | /api/me/volunteer-applications | List own applications |
| GET | /api/volunteer-applications/:id | View own application |
| DELETE | /api/volunteer-applications/:id | Cancel own application |

### Organizer (auth + owner check in service)

| Method | Path | Description |
|--------|------|-------------|
| PUT | /api/events/:id/volunteer/settings | Create/update settings |
| POST | /api/events/:id/volunteer/roles | Add role |
| DELETE | /api/events/:id/volunteer/roles/:roleId | Remove role |
| GET | /api/events/:id/volunteer/applications | List all applications |
| GET | /api/volunteer-applications/:id/admin | Get any application |
| POST | /api/volunteer-applications/:id/approve | Approve |
| POST | /api/volunteer-applications/:id/reject | Reject |

## Business Logic

### Apply Flow (POST /api/events/:id/volunteer/apply)

1. Begin transaction, lock event row (FOR UPDATE)
2. Verify event is published
3. Fetch volunteer_settings — must exist and enabled=true
4. If max_applications is set, count current pending+approved; reject if cap reached
5. Validate role_id belongs to this event
6. Resolve name/email from user profile, allow optional override
7. Validate description is non-empty (required)
8. Insert into volunteer_applications with status='pending'
9. Handle UNIQUE(event_id, user_id) violation → 409

### Approve Flow

1. Begin transaction
2. Get application, verify organizer owns the event
3. Update status to 'approved'
4. Commit

### Reject Flow

1. Get application, verify organizer owns the event
2. Update status to 'rejected'

### Cancel Flow (volunteer)

1. Verify application belongs to requesting user
2. Update status to 'cancelled'

### Delete Role Flow

1. Verify organizer
2. Delete role by id + event_id
3. ON DELETE RESTRICT on applications prevents deletion if apps reference it

## Error Mapping

| Service Error | HTTP Status | Message |
|---|---|---|
| ErrEventNotFound | 404 | event not found |
| ErrEventNotOpen | 409 | event is not open |
| ErrVolunteerSettingsNotFound | 404 | volunteer settings not found |
| ErrVolunteerNotOpen | 409 | volunteer applications are not open |
| ErrVolunteerMaxReached | 409 | application cap reached |
| ErrVolunteerDuplicate | 409 | already applied for this event |
| ErrVolunteerRoleNotFound | 404 | volunteer role not found |
| ErrVolunteerAppNotFound | 404 | volunteer application not found |

## Files

### New

- `supabase/migrations/20260629100000_volunteer_workflow.sql`
- `backend/sql/volunteer.sql`
- `backend/internal/dto/volunteer.go`
- `backend/internal/service/volunteer.go`
- `backend/internal/server/volunteer.go`

### Modified

- `backend/internal/service/errors.go`
- `backend/internal/server/server.go`

### Regenerated (sqlc)

- `backend/internal/db/models.go`
- `backend/internal/db/volunteer.sql.go`
- `backend/internal/db/querier.go`
