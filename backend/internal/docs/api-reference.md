# Pulse API Reference

Complete documentation of every HTTP endpoint in the Pulse backend.

## Conventions

- **Base URL**: `/api` (all protected routes are under the `/api` group)
- **Auth**: Clerk JWT in `Authorization: Bearer <token>` header
- **Content-Type**: `application/json` for request bodies
- **Pagination**: `limit` (default 20, max 100) and `offset` (default 0) query params; responses include `limit`, `offset`, `total`
- **Errors**: All errors return `{"message": "..."}` or `{"message": "...", "errors": [...]}`
- **IDs**: All entity IDs are UUID v4

## Auth Levels

| Level | Middleware | Description |
|---|---|---|
| Public | None | No authentication required |
| Protected | `Auth` + `LoadUser` | Clerk JWT verified, local user loaded into context |
| Organizer | Protected + ownership check | Must be the event's `organizer_id` |
| Event Staff | Protected + `RequireEventStaff` | Organizer OR approved volunteer admin (`is_admin=true`, `status=approved`) |
| Registration Moderation | Protected + `RequireRegistrationModeration` | Organizer always; volunteer admin only if `event.requires_approval=true` |

## Middleware Pipeline

All routes pass through (in order):
1. `Recover` — panic recovery
2. `RequestID` — assigns unique request ID
3. `Logger injection` — attaches `slog.Logger` with request ID to context
4. `CORS` — allows all origins, standard methods + headers
5. `RateLimiter` — 30 req/s, burst 60, 3-minute window

Protected routes additionally pass through:
6. `Auth` — verifies Clerk session JWT from `Authorization` header (local JWKS crypto, no network call)
7. `LoadUser` — loads local `users` row by Clerk `auth_user_id`; on first login, runs onboarding transaction (create user, link guest registrations by email, create default calendar)

---

# Infrastructure

## GET /healthz

Liveness probe.

| | |
|---|---|
| **Auth** | Public |
| **Params** | None |
| **Response 200** | `{"status": "ok"}` |

## GET /api/me

Returns the authenticated user's profile.

| | |
|---|---|
| **Auth** | Protected |
| **Params** | None |
| **Response 200** | `User` object |
| **Errors** | 401 — not authenticated |

### User object

```json
{
  "id": "uuid",
  "auth_user_id": "string",
  "email": "string",
  "username": "string|null",
  "avatar_url": "string|null",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

# Event Discovery (Public)

## GET /api/events

Lists published events for public discovery, ordered by `starts_at` ascending.

| | |
|---|---|
| **Auth** | Public |
| **Query params** | `limit`, `offset`, `category` (one of: technical, cultural, sports, workshop, seminar, hackathon, other), `is_virtual` (`true`/`false`/omit), `q` (title/description search), `starts_after` (RFC3339), `starts_before` (RFC3339) |
| **Response 200** | `EventListResponse` |
| **Errors** | 500 — internal server error |

### EventListResponse

```json
{
  "events": [EventResponse],
  "limit": "int32",
  "offset": "int32",
  "total": "int32"
}
```

### EventResponse

```json
{
  "id": "uuid",
  "organizer_id": "uuid",
  "title": "string",
  "description": "string|null",
  "starts_at": "timestamp",
  "ends_at": "timestamp",
  "capacity": "int32",
  "price_in_paise": "int64",
  "currency": "string",
  "status": "string",
  "slug": "string",
  "category": "string|null",
  "requires_approval": "bool",
  "cover_image_url": "string|null",
  "calendar_id": "uuid|null",
  "is_virtual": "bool",
  "meeting_url": "string|null",
  "venue_name": "string|null",
  "lat": "float64|null",
  "lng": "float64|null",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## GET /api/events/:id

Returns a single event if its status is `published`, `cancelled`, or `completed`. Draft events are hidden.

| | |
|---|---|
| **Auth** | Public |
| **Path params** | `id` — event UUID |
| **Response 200** | `EventResponse` |
| **Errors** | 400 — invalid event id; 404 — event not found or not viewable |

## GET /api/events/slug/:slug

Same as above, but lookup by slug.

| | |
|---|---|
| **Auth** | Public |
| **Path params** | `slug` — event slug string |
| **Response 200** | `EventResponse` |
| **Errors** | 400 — slug required; 404 — event not found or not viewable |

---

# Event Management (Organizer)

## POST /api/events

Creates a new event. The authenticated user becomes the organizer. Event is created with `status='published'`. Slug is generated from the title with a random suffix; on slug collision, retries once with a UUID suffix.

| | |
|---|---|
| **Auth** | Protected |
| **Body** | `CreateEventRequest` |
| **Response 201** | `EventResponse` |
| **Errors** | 400 — validation failed; 500 — internal server error |

### CreateEventRequest

```json
{
  "title": "string (required, 1-200)",
  "description": "string|null (max 5000)",
  "starts_at": "timestamp (required)",
  "ends_at": "timestamp (required, must be after starts_at)",
  "capacity": "int32 (required, min 1)",
  "price_in_paise": "int64 (min 0)",
  "currency": "string (required, len 3)",
  "requires_approval": "bool",
  "cover_image_url": "string|null (valid URL)",
  "calendar_id": "uuid|null",
  "is_virtual": "bool",
  "meeting_url": "string|null (valid URL, required if is_virtual=true)",
  "venue_name": "string|null (max 200, required if is_virtual=false)",
  "lat": "float64|null (latitude, required if is_virtual=false)",
  "lng": "float64|null (longitude, required if is_virtual=false)",
  "category": "string|null (oneof: technical, cultural, sports, workshop, seminar, hackathon, other)"
}
```

**Validation**: Virtual events require `meeting_url`. Physical events require `venue_name`, `lat`, and `lng`. Enforced at struct level.

**Normalization**: Virtual events clear `venue_name`/`lat`/`lng`. Physical events clear `meeting_url`.

## GET /api/me/events

Lists events owned by the authenticated user, ordered by `starts_at` descending.

| | |
|---|---|
| **Auth** | Protected |
| **Query params** | `limit`, `offset`, `include_cancelled` (`true`/`false`) |
| **Response 200** | `EventListResponse` |
| **Errors** | 500 — internal server error |

## PATCH /api/events/:id

Updates an event. Supports two authorization tiers with different field access.

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — event UUID |
| **Body** | `UpdateEventRequest` (all fields optional — partial patch) |
| **Response 200** | `EventResponse` |
| **Errors** | 400 — invalid event id / validation failed; 403 — volunteer admin trying to edit restricted fields; 404 — event not found |

### UpdateEventRequest

```json
{
  "description": "string|null (max 5000)",
  "starts_at": "timestamp|null",
  "ends_at": "timestamp|null",
  "capacity": "int32|null (min 1)",
  "price_in_paise": "int64|null (min 0)",
  "currency": "string|null (len 3)",
  "requires_approval": "bool|null",
  "cover_image_url": "string|null (valid URL)",
  "is_virtual": "bool|null",
  "meeting_url": "string|null (valid URL)",
  "venue_name": "string|null (max 200)",
  "lat": "float64|null (latitude)",
  "lng": "float64|null (longitude)",
  "category": "string|null (oneof)"
}
```

### Authorization tiers

| Actor | Allowed fields |
|---|---|
| **Organizer** | All fields. Merged-state validation: virtual needs meeting_url, physical needs venue+lat+lng, ends_at after starts_at. Normalizes virtual/physical fields. |
| **Volunteer admin** | `description`, `cover_image_url`, `meeting_url`, `venue_name`, `lat`, `lng` only. Attempting to set `starts_at`, `ends_at`, `capacity`, `price_in_paise`, `currency`, `requires_approval`, `is_virtual`, or `category` returns 403. |

**Ownership check**: `ResolveStaffRole` → organizer (full update via `UpdateEvent` SQL with `organizer_id` guard) or volunteer admin (restricted update via `UpdateEventStaffFields` SQL with `organizer_id OR EXISTS admin check` guard). No staff access → 404.

## DELETE /api/events/:id

Soft-cancels an event (`status='cancelled'`). Preserves all registrations, volunteers, kanban, and payment rows. Organizer-only.

| | |
|---|---|
| **Auth** | Protected (organizer only) |
| **Path params** | `id` — event UUID |
| **Response 200** | `EventResponse` (with `status: "cancelled"`) |
| **Errors** | 400 — invalid event id; 404 — event not found or not organizer |

---

# Guest Registration (Public, Token-Based)

## POST /api/events/:id/register/guest

Registers a guest (no user account) for an event. Opens a transaction, locks the event row, checks capacity, generates an access token with 30-day expiry. Status is `approved` (or `pending` if `requires_approval`). On access token collision, regenerates and retries once.

| | |
|---|---|
| **Auth** | Public |
| **Path params** | `id` — event UUID |
| **Body** | `CreateGuestRegistrationRequest` |
| **Response 201** | `RegistrationResponse` |
| **Errors** | 400 — invalid event id / validation; 404 — event not found; 409 — event full / not open / duplicate registration |

### CreateGuestRegistrationRequest

```json
{
  "name": "string (required, 1-200)",
  "email": "string (required, email format)"
}
```

## GET /api/guest-registrations?token=

Looks up a guest registration by its access token. Only returns if token is not expired.

| | |
|---|---|
| **Auth** | Public (token-based) |
| **Query params** | `token` — access token UUID |
| **Response 200** | `RegistrationResponse` |
| **Errors** | 400 — token required / invalid; 404 — registration not found or token expired |

## DELETE /api/guest-registrations?token=

Cancels a guest registration by access token. Only affects rows where `user_id IS NULL` (guests only).

| | |
|---|---|
| **Auth** | Public (token-based) |
| **Query params** | `token` — access token UUID |
| **Response 200** | `RegistrationResponse` (with `status: "cancelled"`) |
| **Errors** | 400 — token required / invalid; 404 — registration not found |

---

# User Registration (Authenticated)

## POST /api/events/:id/register

Registers the authenticated user for an event. Transaction + row lock. Checks capacity. Status is `approved` (or `pending` if `requires_approval`). Name falls back to username, then email local-part. Email falls back to profile email. On access token collision, regenerates and retries.

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — event UUID |
| **Body** | `CreateRegistrationRequest` |
| **Response 201** | `RegistrationResponse` |
| **Errors** | 400 — invalid event id / validation; 404 — event not found; 409 — event full / not open / duplicate registration |

### CreateRegistrationRequest

```json
{
  "name": "string|null (1-200)",
  "email": "string|null (email format)"
}
```

### RegistrationResponse

```json
{
  "id": "uuid",
  "event_id": "uuid",
  "user_id": "uuid|null",
  "name": "string",
  "email": "string",
  "status": "string",
  "access_token": "uuid (present for guest registrations)",
  "token_expires_at": "timestamp|null (present for guest registrations)",
  "checked_in_at": "timestamp|null",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## GET /api/registrations/:id

Fetches one of the user's own registrations. Ownership enforced in SQL (`WHERE id=$1 AND user_id=$2`).

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — registration UUID |
| **Response 200** | `RegistrationResponse` |
| **Errors** | 400 — invalid id; 404 — not found or belongs to another user |

## DELETE /api/registrations/:id

Cancels the user's own registration. No status guard — can cancel from any status (pending/approved/rejected). Ownership enforced in SQL.

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — registration UUID |
| **Response 200** | `RegistrationResponse` (with `status: "cancelled"`) |
| **Errors** | 400 — invalid id; 404 — not found or belongs to another user |

## GET /api/me/registrations

Lists all of the user's own registrations across all events, ordered by `created_at` descending.

| | |
|---|---|
| **Auth** | Protected |
| **Query params** | `limit`, `offset` |
| **Response 200** | `RegistrationListResponse` |
| **Errors** | 500 — internal server error |

### RegistrationListResponse

```json
{
  "registrations": [RegistrationResponse],
  "limit": "int32",
  "offset": "int32",
  "total": "int32"
}
```

---

# Registration Management (Event Staff)

## GET /api/events/:id/registrations

Lists registrations for an event. Both organizer and volunteer admin can view all registrations.

| | |
|---|---|
| **Auth** | Event Staff |
| **Path params** | `id` — event UUID |
| **Query params** | `limit`, `offset` |
| **Response 200** | `RegistrationListResponse` |
| **Errors** | 400 — invalid event id; 404 — event not found or no staff access |

**Authorization**: `RequireEventStaff` (organizer OR volunteer admin). SQL re-enforces: `WHERE event_id=$1 AND (organizer_id=$2 OR EXISTS admin check)`.

## GET /api/registrations/:id/admin

Fetches a single registration as staff. Ownership enforced in SQL.

| | |
|---|---|
| **Auth** | Event Staff |
| **Path params** | `id` — registration UUID |
| **Response 200** | `RegistrationResponse` |
| **Errors** | 400 — invalid id; 404 — not found or no staff access |

## POST /api/registrations/:id/approve

Approves a pending registration. Idempotent — if already approved, returns existing.

| | |
|---|---|
| **Auth** | Registration Moderation |
| **Path params** | `id` — registration UUID |
| **Response 200** | `RegistrationResponse` (with `status: "approved"`) |
| **Errors** | 400 — invalid id; 403 — volunteer admin on non-approval-gated event; 404 — not found or no staff access; 409 — event full |

**Authorization**: Three layers of defense:
1. `GetRegistrationByIDAdmin` pre-check (broad: organizer OR admin)
2. `RequireRegistrationModeration`: organizer → always; volunteer admin → only if `event.requires_approval=true` → 403 otherwise
3. SQL: `WHERE ... AND (organizer_id=$2 OR (requires_approval=true AND EXISTS admin check))`

**Capacity check**: Inside transaction with event row locked. `validateCapacityForApproval`: pending→approved checks `count > capacity`; rejected/cancelled→approved checks `count >= capacity`.

## POST /api/registrations/:id/reject

Rejects a pending registration. Same authorization as approve.

| | |
|---|---|
| **Auth** | Registration Moderation |
| **Path params** | `id` — registration UUID |
| **Response 200** | `RegistrationResponse` (with `status: "rejected"`) |
| **Errors** | 400 — invalid id; 403 — volunteer admin on non-approval-gated event; 404 — not found or no staff access |

## PATCH /api/registrations/:id/status

Sets a registration to an arbitrary valid status. **Organizer-only** — volunteer admins get 404.

| | |
|---|---|
| **Auth** | Protected (organizer only) |
| **Path params** | `id` — registration UUID |
| **Body** | `UpdateRegistrationStatusRequest` |
| **Response 200** | `RegistrationResponse` |
| **Errors** | 400 — invalid id / validation; 404 — not found or not organizer; 409 — event full (when approving) |

### UpdateRegistrationStatusRequest

```json
{
  "status": "string (required, oneof: pending, approved, rejected, cancelled)"
}
```

**Note**: This is the one endpoint where volunteer admins get 404 instead of access. This is intentional — the generic status endpoint allows arbitrary transitions, while approve/reject are the controlled transitions for volunteer admins.

**Capacity check**: When transitioning to `approved`, locks event row and checks capacity inside a transaction. Idempotent if same status.

## POST /api/registrations/:id/check-in

Checks in an attendee by access token. Both organizer and volunteer admin can check in (no `requires_approval` guard — check-in is a venue operation). Idempotent — if already checked in, returns existing registration.

| | |
|---|---|
| **Auth** | Event Staff |
| **Path params** | `id` — registration UUID |
| **Body** | `CheckInRequest` |
| **Response 200** | `RegistrationResponse` (with `checked_in_at` set) |
| **Errors** | 400 — invalid id / validation; 404 — not found / wrong token; 409 — event not published |

### CheckInRequest

```json
{
  "access_token": "uuid (required)"
}
```

**Authorization**: SQL: `WHERE ... AND (organizer_id=$2 OR EXISTS admin check) AND e.status='published' AND r.access_token=$3 AND r.checked_in_at IS NULL`.

**ErrNoRows disambiguation**: Re-fetches via `GetRegistrationByIDAdmin` → token mismatch → 404; already checked in → 200 (idempotent); event not published → 409.

---

# Volunteer — Public Info

## GET /api/events/:id/volunteer/settings

Returns volunteer settings for an event.

| | |
|---|---|
| **Auth** | Public |
| **Path params** | `id` — event UUID |
| **Response 200** | `VolunteerSettingsResponse` |
| **Errors** | 400 — invalid event id; 404 — volunteer settings not found |

### VolunteerSettingsResponse

```json
{
  "event_id": "uuid",
  "enabled": "bool",
  "description": "string|null",
  "max_applications": "int32|null",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## GET /api/events/:id/volunteer/roles

Lists all volunteer roles for an event, ordered by name.

| | |
|---|---|
| **Auth** | Public |
| **Path params** | `id` — event UUID |
| **Response 200** | `[VolunteerRoleResponse]` (empty array if none) |
| **Errors** | 400 — invalid event id; 500 — internal server error |

### VolunteerRoleResponse

```json
{
  "id": "uuid",
  "event_id": "uuid",
  "name": "string",
  "is_admin": "bool",
  "created_at": "timestamp"
}
```

---

# Volunteer — User Applications (Authenticated)

## POST /api/events/:id/volunteer/apply

Applies to volunteer for a specific role. Transaction + row lock on event. Checks: event published, volunteering enabled, cap not reached, role belongs to event, no duplicate active application. Re-application after rejection works (rejected rows excluded from unique index).

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — event UUID |
| **Body** | `CreateVolunteerApplicationRequest` |
| **Response 201** | `VolunteerApplicationResponse` |
| **Errors** | 400 — invalid event id / validation; 404 — event/role not found; 409 — not open / cap reached / duplicate active application |

### CreateVolunteerApplicationRequest

```json
{
  "role_id": "uuid (required)",
  "name": "string|null (1-200)",
  "email": "string|null (email format)",
  "phone": "string|null (max 20)",
  "description": "string (required, 1-5000)"
}
```

### VolunteerApplicationResponse

```json
{
  "id": "uuid",
  "event_id": "uuid",
  "user_id": "uuid",
  "role_id": "uuid",
  "role_name": "string",
  "is_admin": "bool",
  "name": "string",
  "email": "string",
  "phone": "string|null",
  "description": "string",
  "status": "string (pending, approved, rejected, cancelled)",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## GET /api/me/event-staff

Lists events where the user is an approved volunteer admin (`status=approved AND is_admin=true`).

| | |
|---|---|
| **Auth** | Protected |
| **Query params** | `limit`, `offset` |
| **Response 200** | `EventStaffListResponse` |
| **Errors** | 500 — internal server error |

### EventStaffListResponse

```json
{
  "events": [EventStaffItemResponse],
  "limit": "int32",
  "offset": "int32",
  "total": "int32"
}
```

### EventStaffItemResponse

```json
{
  "event_id": "uuid",
  "title": "string",
  "slug": "string",
  "starts_at": "timestamp",
  "ends_at": "timestamp",
  "event_status": "string",
  "application_id": "uuid",
  "role_id": "uuid",
  "role_name": "string",
  "is_admin": "bool",
  "approved_at": "timestamp"
}
```

## GET /api/me/volunteer-applications

Lists all of the user's own volunteer applications across all events, with role info. All statuses included.

| | |
|---|---|
| **Auth** | Protected |
| **Query params** | `limit`, `offset` |
| **Response 200** | `VolunteerApplicationListResponse` |
| **Errors** | 500 — internal server error |

### VolunteerApplicationListResponse

```json
{
  "applications": [VolunteerApplicationResponse],
  "limit": "int32",
  "offset": "int32",
  "total": "int32"
}
```

## GET /api/volunteer-applications/:id

Fetches one of the user's own applications. Ownership enforced in SQL (`WHERE id=$1 AND user_id=$2`).

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — application UUID |
| **Response 200** | `VolunteerApplicationResponse` |
| **Errors** | 400 — invalid id; 404 — not found or belongs to another user |

## DELETE /api/volunteer-applications/:id

Cancels the user's own **pending** application. Sets `status='cancelled'`, freeing the unique index slot so they can re-apply.

| | |
|---|---|
| **Auth** | Protected |
| **Path params** | `id` — application UUID |
| **Response 200** | `VolunteerApplicationResponse` (with `status: "cancelled"`) |
| **Errors** | 400 — invalid id; 404 — not found or belongs to another user; 409 — only pending applications can be cancelled |

---

# Volunteer — Management (Organizer Only)

## PUT /api/events/:id/volunteer/settings

Creates or updates volunteer settings. Atomic upsert (`ON CONFLICT (event_id) DO UPDATE`). Organizer-only.

| | |
|---|---|
| **Auth** | Protected (organizer only) |
| **Path params** | `id` — event UUID |
| **Body** | `UpsertVolunteerSettingsRequest` |
| **Response 200** | `VolunteerSettingsResponse` |
| **Errors** | 400 — invalid event id / validation; 404 — event not found or not organizer |

### UpsertVolunteerSettingsRequest

```json
{
  "enabled": "bool",
  "description": "string|null (max 2000)",
  "max_applications": "int32|null (min 1, null = unlimited)"
}
```

## POST /api/events/:id/volunteer/roles

Creates a volunteer role with a name and `is_admin` flag. Organizer-only.

| | |
|---|---|
| **Auth** | Protected (organizer only) |
| **Path params** | `id` — event UUID |
| **Body** | `CreateVolunteerRoleRequest` |
| **Response 201** | `VolunteerRoleResponse` |
| **Errors** | 400 — invalid event id / validation; 404 — event not found or not organizer; 409 — role name already exists for this event |

### CreateVolunteerRoleRequest

```json
{
  "name": "string (required, 1-100)",
  "is_admin": "bool"
}
```

## DELETE /api/events/:id/volunteer/roles/:roleId

Deletes a volunteer role. Validates role belongs to the event and caller is the organizer. Blocked if the role has applications (FK `ON DELETE RESTRICT`).

| | |
|---|---|
| **Auth** | Protected (organizer only) |
| **Path params** | `id` — event UUID, `roleId` — role UUID |
| **Response 200** | `VolunteerRoleResponse` |
| **Errors** | 400 — invalid id(s); 404 — role/event not found or not organizer; 409 — role has applications |

---

# Volunteer — Management (Event Staff)

## GET /api/events/:id/volunteer/applications

Lists all volunteer applications for an event. Both organizer and volunteer admin can see all applications.

| | |
|---|---|
| **Auth** | Event Staff |
| **Path params** | `id` — event UUID |
| **Query params** | `limit`, `offset` |
| **Response 200** | `VolunteerApplicationListResponse` |
| **Errors** | 400 — invalid event id; 404 — event not found or no staff access |

## GET /api/volunteer-applications/:id/admin

Fetches any single application as staff. Ownership enforced in SQL.

| | |
|---|---|
| **Auth** | Event Staff |
| **Path params** | `id` — application UUID |
| **Response 200** | `VolunteerApplicationResponse` |
| **Errors** | 400 — invalid id; 404 — not found or no staff access |

## POST /api/volunteer-applications/:id/approve

Approves a pending application → `status=approved` and inserts into the `volunteers` roster table. Both operations run in a single transaction.

| | |
|---|---|
| **Auth** | Event Staff (with admin-role restriction) |
| **Path params** | `id` — application UUID |
| **Response 200** | `VolunteerApplicationResponse` (with `status: "approved"`) |
| **Errors** | 400 — invalid id; 403 — volunteer admin approving an admin-role application; 404 — not found or no staff access; 409 — not pending |

**Authorization**: SQL (source of truth): `WHERE ... AND (e.organizer_id=$2 OR (target.is_admin=false AND EXISTS admin check)) AND va.status='pending'`.

- **Organizer**: can approve any role.
- **Volunteer admin**: can only approve **non-admin** role applications (`target.is_admin=false`). Approving an admin-role application returns 403.

**Transaction**: `ApproveVolunteerApplication` → `GetVolunteerRoleByID` → `InsertVolunteer` (idempotent via `ON CONFLICT DO NOTHING`). If roster insert fails, approval is rolled back.

**ErrNoRows disambiguation**: `volunteerModerationError` re-fetches (broad) → if pending: `IsEventOrganizer` check → if `!isOrg && existing.IsAdmin` → 403; else → 404.

## POST /api/volunteer-applications/:id/reject

Rejects a pending application → `status=rejected`. Frees the unique index slot so the user can re-apply. Same admin-role restriction as approve.

| | |
|---|---|
| **Auth** | Event Staff (with admin-role restriction) |
| **Path params** | `id` — application UUID |
| **Response 200** | `VolunteerApplicationResponse` (with `status: "rejected"`) |
| **Errors** | 400 — invalid id; 403 — volunteer admin rejecting an admin-role application; 404 — not found or no staff access; 409 — not pending |

## GET /api/events/:id/volunteers

Returns the confirmed volunteer roster from the `volunteers` table, joined with `users` for profile info. Ordered by `created_at` descending.

| | |
|---|---|
| **Auth** | Event Staff |
| **Path params** | `id` — event UUID |
| **Query params** | `limit`, `offset` |
| **Response 200** | `VolunteerListResponse` |
| **Errors** | 400 — invalid event id; 404 — event not found or no staff access; 500 — internal server error |

**Authorization**: SQL: `WHERE v.event_id=$1 AND (e.organizer_id=$2 OR EXISTS admin check)`.

### VolunteerListResponse

```json
{
  "volunteers": [VolunteerResponse],
  "limit": "int32",
  "offset": "int32",
  "total": "int32"
}
```

### VolunteerResponse

```json
{
  "id": "uuid",
  "event_id": "uuid",
  "user_id": "uuid",
  "role": "string",
  "email": "string",
  "username": "string|null",
  "avatar_url": "string|null",
  "created_at": "timestamp"
}
```

---

# Capability Matrix

| Capability | Organizer | Volunteer Admin | Regular User | Public |
|---|:---:|:---:|:---:|:---:|
| View published events | ✅ | ✅ | ✅ | ✅ |
| Create events | ✅ | ✅ | ✅ | ❌ |
| Edit own events (all fields) | ✅ | ❌ | ❌ | ❌ |
| Edit events (restricted fields) | ✅ | ✅ | ❌ | ❌ |
| Cancel own events | ✅ | ❌ | ❌ | ❌ |
| Register for events | ✅ | ✅ | ✅ | ✅ (guest) |
| Cancel own registration | ✅ | ✅ | ✅ | ✅ (guest, by token) |
| List event registrations | ✅ | ✅ | ❌ | ❌ |
| Approve/reject registrations | ✅ | ✅ (gated events only) | ❌ | ❌ |
| Set arbitrary registration status | ✅ | ❌ | ❌ | ❌ |
| Check in attendees | ✅ | ✅ | ❌ | ❌ |
| View volunteer settings/roles | ✅ | ✅ | ✅ | ✅ |
| Configure volunteer settings | ✅ | ❌ | ❌ | ❌ |
| Create/delete volunteer roles | ✅ | ❌ | ❌ | ❌ |
| Apply to volunteer | ✅ | ✅ | ✅ | ❌ |
| Cancel own volunteer application | ✅ | ✅ | ✅ | ❌ |
| View own volunteer applications | ✅ | ✅ | ✅ | ❌ |
| View events I help manage | ✅ | ✅ | ✅ (own only) | ❌ |
| List event's volunteer applications | ✅ | ✅ | ❌ | ❌ |
| Approve/reject volunteer apps (non-admin role) | ✅ | ✅ | ❌ | ❌ |
| Approve/reject volunteer apps (admin role) | ✅ | ❌ (403) | ❌ | ❌ |
| List confirmed volunteer roster | ✅ | ✅ | ❌ | ❌ |
