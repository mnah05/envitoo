# Volunteer Workflow — Implementation Details

## Overview

A complete volunteer application system where event hosts can configure volunteer roles, collect applications from signed-in users, and manage approvals/rejections. Host-defined roles support an `is_admin` flag for future permission-based features.

---

## Database Schema

### Migration File

**`supabase/migrations/20260629100000_volunteer_workflow.sql`**

```sql
CREATE TABLE volunteer_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL UNIQUE REFERENCES events (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    max_applications INT4,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE volunteer_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, name)
);

CREATE INDEX idx_volunteer_roles_event_id ON volunteer_roles (event_id);

CREATE TABLE volunteer_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES volunteer_roles (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, user_id)
);

CREATE INDEX idx_volunteer_apps_event_id ON volunteer_applications (event_id);
CREATE INDEX idx_volunteer_apps_user_id ON volunteer_applications (user_id);
CREATE INDEX idx_volunteer_apps_status ON volunteer_applications (status);
```

### Table Details

| Table | Purpose | Key Constraints |
|-------|---------|----------------|
| `volunteer_settings` | Per-event configuration (enabled toggle, host description, max applications cap) | UNIQUE(event_id) |
| `volunteer_roles` | Host-defined roles with names and optional admin flag | UNIQUE(event_id, name), ON DELETE RESTRICT on applications |
| `volunteer_applications` | Signed-in user applications with personal details and work description | UNIQUE(event_id, user_id) |

---

## SQL Queries

**File: `backend/sql/volunteer.sql`**

17 queries generated via sqlc:

### Settings (2 queries)

```sql
-- UpsertVolunteerSettings :one
INSERT INTO volunteer_settings (event_id, enabled, description, max_applications, created_at, updated_at)
VALUES ($1, $2, $3, $4, now(), now())
ON CONFLICT (event_id)
DO UPDATE SET enabled = EXCLUDED.enabled, description = EXCLUDED.description,
              max_applications = EXCLUDED.max_applications, updated_at = now()
RETURNING *;

-- GetVolunteerSettingsByEvent :one
SELECT * FROM volunteer_settings WHERE event_id = $1;
```

### Roles (4 queries)

```sql
-- CreateVolunteerRole :one
INSERT INTO volunteer_roles (event_id, name, is_admin, created_at)
VALUES ($1, $2, $3, now()) RETURNING *;

-- GetVolunteerRoleByID :one
SELECT * FROM volunteer_roles WHERE id = $1;

-- DeleteVolunteerRole :one
DELETE FROM volunteer_roles WHERE id = $1 AND event_id = $2 RETURNING *;

-- ListVolunteerRolesByEvent :many
SELECT * FROM volunteer_roles WHERE event_id = $1 ORDER BY name;
```

### Applications — Volunteer (5 queries)

```sql
-- CreateVolunteerApplication :one
INSERT INTO volunteer_applications (...)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now()) RETURNING *;

-- GetVolunteerApplicationWithRole :one
SELECT va.*, vr.name AS role_name, vr.is_admin AS is_admin
FROM volunteer_applications va JOIN volunteer_roles vr ON va.role_id = vr.id WHERE va.id = $1;

-- GetVolunteerApplicationByID :one
SELECT va.*, vr.name AS role_name, vr.is_admin AS is_admin
FROM volunteer_applications va JOIN volunteer_roles vr ON va.role_id = vr.id
WHERE va.id = $1 AND va.user_id = $2;

-- ListMyVolunteerApplications :many
SELECT va.*, vr.name AS role_name, vr.is_admin AS is_admin
FROM volunteer_applications va JOIN volunteer_roles vr ON va.role_id = vr.id
WHERE va.user_id = $1 ORDER BY va.created_at DESC LIMIT $2 OFFSET $3;

-- CountMyVolunteerApplications :one
SELECT COUNT(*)::int4 FROM volunteer_applications WHERE user_id = $1;

-- CancelVolunteerApplication :one
UPDATE volunteer_applications SET status = 'cancelled', updated_at = now()
WHERE id = $1 AND user_id = $2 RETURNING *;
```

### Applications — Organizer (5 queries)

```sql
-- GetVolunteerApplicationByIDAdmin :one
SELECT va.*, vr.name AS role_name, vr.is_admin AS is_admin
FROM volunteer_applications va JOIN volunteer_roles vr ON va.role_id = vr.id
JOIN events e ON e.id = va.event_id WHERE va.id = $1 AND e.organizer_id = $2;

-- ListEventVolunteerApplications :many
SELECT va.*, vr.name AS role_name, vr.is_admin AS is_admin
FROM volunteer_applications va JOIN volunteer_roles vr ON va.role_id = vr.id
JOIN events e ON e.id = va.event_id
WHERE va.event_id = $1 AND e.organizer_id = $2 ORDER BY va.created_at DESC LIMIT $3 OFFSET $4;

-- CountEventVolunteerApplications :one
SELECT COUNT(*)::int4 FROM volunteer_applications va
JOIN events e ON e.id = va.event_id WHERE va.event_id = $1 AND e.organizer_id = $2;

-- CountVolunteerApplicationsByEvent :one
SELECT COUNT(*)::int4 FROM volunteer_applications WHERE event_id = $1;

-- ApproveVolunteerApplication :one
UPDATE volunteer_applications va SET status = 'approved', updated_at = now()
FROM events e WHERE va.id = $1 AND e.id = va.event_id AND e.organizer_id = $2 RETURNING va.*;

-- RejectVolunteerApplication :one
UPDATE volunteer_applications va SET status = 'rejected', updated_at = now()
FROM events e WHERE va.id = $1 AND e.id = va.event_id AND e.organizer_id = $2 RETURNING va.*;
```

---

## DTOs

**File: `backend/internal/dto/volunteer.go`**

### Settings

```go
type UpsertVolunteerSettingsRequest struct {
    Enabled         bool    `json:"enabled"`
    Description     *string `json:"description" validate:"omitempty,max=2000"`
    MaxApplications *int32  `json:"max_applications" validate:"omitempty,min=1"`
}

type VolunteerSettingsResponse struct {
    EventID         uuid.UUID `json:"event_id"`
    Enabled         bool      `json:"enabled"`
    Description     *string   `json:"description,omitempty"`
    MaxApplications *int32    `json:"max_applications,omitempty"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

func NewVolunteerSettingsResponse(s db.VolunteerSetting) VolunteerSettingsResponse
```

### Roles

```go
type CreateVolunteerRoleRequest struct {
    Name    string `json:"name" validate:"required,min=1,max=100"`
    IsAdmin bool   `json:"is_admin"`
}

type VolunteerRoleResponse struct {
    ID        uuid.UUID `json:"id"`
    EventID   uuid.UUID `json:"event_id"`
    Name      string    `json:"name"`
    IsAdmin   bool      `json:"is_admin"`
    CreatedAt time.Time `json:"created_at"`
}

func NewVolunteerRoleResponse(r db.VolunteerRole) VolunteerRoleResponse
```

### Applications

```go
type CreateVolunteerApplicationRequest struct {
    RoleID      uuid.UUID `json:"role_id" validate:"required"`
    Name        *string   `json:"name" validate:"omitempty,min=1,max=200"`
    Email       *string   `json:"email" validate:"omitempty,email"`
    Phone       *string   `json:"phone" validate:"omitempty,max=20"`
    Description string    `json:"description" validate:"required,min=1,max=5000"`
}

type VolunteerApplicationResponse struct {
    ID          uuid.UUID `json:"id"`
    EventID     uuid.UUID `json:"event_id"`
    UserID      uuid.UUID `json:"user_id"`
    RoleID      uuid.UUID `json:"role_id"`
    RoleName    string    `json:"role_name"`
    IsAdmin     bool      `json:"is_admin"`
    Name        string    `json:"name"`
    Email       string    `json:"email"`
    Phone       *string   `json:"phone,omitempty"`
    Description string    `json:"description"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type VolunteerApplicationListResponse struct {
    Applications []VolunteerApplicationResponse `json:"applications"`
    Limit        int32                          `json:"limit"`
    Offset       int32                          `json:"offset"`
    Total        int32                          `json:"total"`
}
```

---

## Service Layer

**File: `backend/internal/service/volunteer.go`**

### Constructor

```go
func NewVolunteerService(pool *pgxpool.Pool, queries *db.Queries, log *slog.Logger) *VolunteerService
```

### Methods (13 total)

#### Settings

| Method | Signature | Description |
|--------|-----------|-------------|
| `UpsertSettings` | `(ctx, eventID, organizerID, req) → (db.VolunteerSetting, error)` | Creates or updates volunteer settings for an event. Verifies the requester is the event organizer. Uses `ON CONFLICT (event_id) DO UPDATE` for atomic upsert. |
| `GetSettings` | `(ctx, eventID) → (db.VolunteerSetting, error)` | Retrieves volunteer settings for an event. Returns `ErrVolunteerSettingsNotFound` if not found. |

#### Roles

| Method | Signature | Description |
|--------|-----------|-------------|
| `CreateRole` | `(ctx, eventID, organizerID, req) → (db.VolunteerRole, error)` | Adds a volunteer role to an event. Verifies organizer ownership. |
| `DeleteRole` | `(ctx, roleID, organizerID) → (db.VolunteerRole, error)` | Removes a role. First fetches the role to get its event_id, then verifies organizer. Uses `ON DELETE RESTRICT` on applications table to prevent deletion if applications exist. |
| `ListRolesByEvent` | `(ctx, eventID) → ([]db.VolunteerRole, error)` | Returns all roles for an event. Public read. |

#### Applications — Volunteer

| Method | Signature | Description |
|--------|-----------|-------------|
| `Apply` | `(ctx, user, eventID, req) → (dto.VolunteerApplicationResponse, error)` | **Core apply flow:** 1) Begin transaction, 2) Lock event row with `FOR UPDATE`, 3) Verify event is published, 4) Fetch settings (must exist + enabled), 5) Check max_applications cap via `CountVolunteerApplicationsByEvent`, 6) Verify role belongs to event, 7) Resolve name/email from user profile with optional override, 8) Insert application with `pending` status, 9) Handle `UNIQUE` violation as `ErrVolunteerDuplicate`. |
| `ListByUser` | `(ctx, userID, params) → ([]dto.VolunteerApplicationResponse, int32, error)` | Paginated list of own applications. Default limit 20, max 100. Returns items + total count. |
| `GetByID` | `(ctx, appID, userID) → (dto.VolunteerApplicationResponse, error)` | Returns single own application. Uses `GetVolunteerApplicationByID` which filters by user_id. |
| `Cancel` | `(ctx, appID, userID) → (dto.VolunteerApplicationResponse, error)` | Cancels own application by setting status to `cancelled`. Returns updated application with role data. |

#### Applications — Organizer

| Method | Signature | Description |
|--------|-----------|-------------|
| `ListByEvent` | `(ctx, eventID, organizerID, params) → ([]dto.VolunteerApplicationResponse, int32, error)` | Paginated list of all applications for an event. Verifies organizer ownership before querying. |
| `GetByIDAsAdmin` | `(ctx, appID, organizerID) → (dto.VolunteerApplicationResponse, error)` | Returns any application for the organizer's event. Uses `GetVolunteerApplicationByIDAdmin` which joins events table. |
| `Approve` | `(ctx, appID, organizerID) → (dto.VolunteerApplicationResponse, error)` | Sets application status to `approved`. Uses `ApproveVolunteerApplication` query which verifies organizer via events JOIN. Returns updated application with role data. |
| `Reject` | `(ctx, appID, organizerID) → (dto.VolunteerApplicationResponse, error)` | Sets application status to `rejected`. Same organizer verification pattern as Approve. |

### Helper Methods

| Method | Description |
|--------|-------------|
| `getApplicationWithRole` | Fetches a single application by ID with joined role data (role_name, is_admin). |
| `lockEventForVolunteer` | Locks event row with `FOR UPDATE`, verifies it exists and is published. Reuses existing `GetEventByIDForUpdate` query. |

---

## HTTP Handlers

**File: `backend/internal/server/volunteer.go`**

12 handlers following the same pattern as `registration.go`:

1. Extract logger from context
2. Parse UUID path params
3. Bind & validate request body (if POST/PUT)
4. Extract user from context (for auth routes)
5. Call service method
6. Map service errors to HTTP errors
7. Return JSON response

### Handler Table

| Handler | Route | Auth | Description |
|---------|-------|------|-------------|
| `GetVolunteerSettings` | `GET /api/events/:id/volunteer/settings` | Public | Returns enabled, description, max_applications |
| `ListVolunteerRoles` | `GET /api/events/:id/volunteer/roles` | Public | Returns list of host-defined roles |
| `CreateVolunteerApplication` | `POST /api/events/:id/volunteer/apply` | User | Submits application with role, description, optional phone/name/email override |
| `ListMyVolunteerApplications` | `GET /api/me/volunteer-applications` | User | Paginated list of own applications |
| `GetMyVolunteerApplication` | `GET /api/volunteer-applications/:id` | User | Single own application with full details |
| `CancelVolunteerApplication` | `DELETE /api/volunteer-applications/:id` | User | Cancels own application |
| `UpsertVolunteerSettings` | `PUT /api/events/:id/volunteer/settings` | Organizer | Creates or updates settings |
| `CreateVolunteerRole` | `POST /api/events/:id/volunteer/roles` | Organizer | Adds a new role |
| `DeleteVolunteerRole` | `DELETE /api/events/:id/volunteer/roles/:roleId` | Organizer | Removes a role (RESTRICT if apps exist) |
| `ListEventVolunteerApplications` | `GET /api/events/:id/volunteer/applications` | Organizer | Paginated list of all applications |
| `GetVolunteerApplicationAsAdmin` | `GET /api/volunteer-applications/:id/admin` | Organizer | Any application detail |
| `ApproveVolunteerApplication` | `POST /api/volunteer-applications/:id/approve` | Organizer | Approves application |
| `RejectVolunteerApplication` | `POST /api/volunteer-applications/:id/reject` | Organizer | Rejects application |

### Error Mapping

```go
func volunteerApplyError(err error) *echo.HTTPError {
    switch {
    case errors.Is(err, service.ErrEventNotFound):
        return echo.NewHTTPError(404, "event not found")
    case errors.Is(err, service.ErrEventNotOpen):
        return echo.NewHTTPError(409, "event is not open for applications")
    case errors.Is(err, service.ErrVolunteerNotOpen):
        return echo.NewHTTPError(409, "volunteer applications are not open")
    case errors.Is(err, service.ErrVolunteerMaxReached):
        return echo.NewHTTPError(409, "volunteer application cap reached")
    case errors.Is(err, service.ErrVolunteerDuplicate):
        return echo.NewHTTPError(409, "already applied for this event")
    case errors.Is(err, service.ErrVolunteerRoleNotFound):
        return echo.NewHTTPError(404, "role not found")
    default:
        return nil
    }
}
```

---

## Service Errors

**File: `backend/internal/service/errors.go`**

```go
var (
    ErrEventNotFound             = errors.New("event not found")
    ErrRegistrationNotFound      = errors.New("registration not found")
    ErrInvalidEvent              = errors.New("invalid event")
    ErrConflict                  = errors.New("registration already exists")
    ErrEventFull                 = errors.New("event is full")
    ErrEventNotOpen              = errors.New("event not open for registration")
    ErrVolunteerSettingsNotFound = errors.New("volunteer settings not found")
    ErrVolunteerNotOpen          = errors.New("volunteer applications are not open")
    ErrVolunteerMaxReached       = errors.New("volunteer application cap reached")
    ErrVolunteerDuplicate        = errors.New("already applied for this event")
    ErrVolunteerRoleNotFound     = errors.New("volunteer role not found")
    ErrVolunteerAppNotFound      = errors.New("volunteer application not found")
)
```

---

## Server Integration

**File: `backend/internal/server/server.go`**

### Server struct addition

```go
type Server struct {
    // ... existing fields
    volunteers *service.VolunteerService
}
```

### Constructor addition

```go
func New(...) *Server {
    return &Server{
        // ... existing fields
        volunteers: service.NewVolunteerService(pool, queries, log),
    }
}
```

### Route wiring

```go
// Public volunteer info (no auth required)
e.GET("/api/events/:id/volunteer/settings", s.GetVolunteerSettings)
e.GET("/api/events/:id/volunteer/roles", s.ListVolunteerRoles)

// Volunteer application routes (auth user)
protected.POST("/events/:id/volunteer/apply", s.CreateVolunteerApplication)
protected.GET("/me/volunteer-applications", s.ListMyVolunteerApplications)
protected.GET("/volunteer-applications/:id", s.GetMyVolunteerApplication)
protected.DELETE("/volunteer-applications/:id", s.CancelVolunteerApplication)

// Organizer volunteer management
protected.PUT("/events/:id/volunteer/settings", s.UpsertVolunteerSettings)
protected.POST("/events/:id/volunteer/roles", s.CreateVolunteerRole)
protected.DELETE("/events/:id/volunteer/roles/:roleId", s.DeleteVolunteerRole)
protected.GET("/events/:id/volunteer/applications", s.ListEventVolunteerApplications)
protected.GET("/volunteer-applications/:id/admin", s.GetVolunteerApplicationAsAdmin)
protected.POST("/volunteer-applications/:id/approve", s.ApproveVolunteerApplication)
protected.POST("/volunteer-applications/:id/reject", s.RejectVolunteerApplication)
```

---

## sqlc Generated Models

**File: `backend/internal/db/models.go`**

Three new structs generated:

```go
type VolunteerApplication struct {
    ID          uuid.UUID
    EventID     uuid.UUID
    UserID      uuid.UUID
    RoleID      uuid.UUID
    Name        string
    Email       string
    Phone       *string
    Description string
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type VolunteerRole struct {
    ID        uuid.UUID
    EventID   uuid.UUID
    Name      string
    IsAdmin   bool
    CreatedAt time.Time
}

type VolunteerSetting struct {
    ID              uuid.UUID
    EventID         uuid.UUID
    Enabled         bool
    Description     *string
    MaxApplications *int32
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

---

## Business Rules

### Apply Flow

1. **Transaction required** — uses `BEGIN` + `FOR UPDATE` on event row to prevent race conditions
2. **Event must be published** — rejects with `ErrEventNotOpen` if status != "published"
3. **Settings must exist and be enabled** — rejects with `ErrVolunteerNotOpen` if no settings or `enabled=false`
4. **Max applications cap** — if `max_applications` is set, counts current applications (all statuses except cancelled/rejected). Rejects with `ErrVolunteerMaxReached` if cap reached.
5. **Role must belong to event** — rejects with `ErrVolunteerRoleNotFound` if role_id doesn't match event
6. **One application per user per event** — `UNIQUE(event_id, user_id)` constraint. Rejects with `ErrVolunteerDuplicate`.
7. **Name/email fallbacks** — if not provided, uses user's `Username` or email local-part for name, and user's `Email` for email

### Approval Flow

1. Verifies organizer via `events` table JOIN in SQL query
2. Updates status to `approved`
3. Returns application with joined role data

### Role Deletion

- `ON DELETE RESTRICT` on `volunteer_applications.role_id` prevents deleting roles that have applications
- Organizer must own the event

### Admin Roles (implemented)

Approved volunteers on a role with `is_admin = true` are **event staff** via `EventAccessService` ([`internal/service/event_access.go`](../internal/service/event_access.go)). SQL staff predicates enforce access in addition to service checks.

| Action | Organizer | Volunteer admin |
|--------|-----------|-------------------|
| Check-in (`POST /registrations/:id/check-in` + `access_token`) | Yes | Yes |
| List/get registrations | Yes | Yes |
| Approve/reject registrations | Yes | Only when `event.requires_approval = true` |
| List/get/approve/reject volunteer applications | Yes | Yes |
| `PATCH /events/:id` | Full body | `description`, `venue_name`, `lat`, `lng`, `meeting_url`, `cover_image_url` only |

Volunteer settings and role CRUD remain organizer-only.

#### Bootstrapping the first volunteer admin

1. Host: `POST /api/events/:id/volunteer/roles` with `{ "name": "...", "is_admin": true }`
2. User: `POST /api/events/:id/volunteer/apply` (selects the admin role)
3. **Host** approves: `POST /api/volunteer-applications/:id/approve` — required for the initial grant; the applicant is not staff until this step
4. User now has volunteer-admin powers on that event

After step 4, that volunteer admin can also list and approve/reject **other** volunteer applications on the same event (same endpoints as the host).

---

## File Summary

| Type | File | Lines |
|------|------|-------|
| Spec | `backend/docs/volunteer-workflow.md` | — |
| Migration | `backend/supabase/migrations/20260629100000_volunteer_workflow.sql` | ~50 |
| SQL | `backend/sql/volunteer.sql` | ~135 |
| DTO | `backend/internal/dto/volunteer.go` | ~90 |
| Service | `backend/internal/service/volunteer.go` | ~500 |
| Handlers | `backend/internal/server/volunteer.go` | ~280 |
| Modified | `backend/internal/service/errors.go` | +6 errors |
| Modified | `backend/internal/server/server.go` | +service + routes |
| Regenerated | `backend/internal/db/models.go` | +3 structs |
| Regenerated | `backend/internal/db/volunteer.sql.go` | ~650 |
| Regenerated | `backend/internal/db/querier.go` | +17 interface methods |

---

## Build Verification

```bash
cd backend && go build ./cmd/server    # compiles clean
cd backend && go vet ./...             # no issues
cd backend && go test ./...            # all pass (no test files in new code)
```

---

## Next Steps (Future)

1. **Frontend integration** — Event dashboard staff views, QR check-in UI, volunteer admin flows
3. **Contact info** — Add `volunteer_contact_email` and `volunteer_contact_phone` to events table for host-provided contact
4. **Email notifications** — Send emails on application submit/approve/reject
5. **Volunteer capacity per role** — Track max volunteers per role instead of just per event
