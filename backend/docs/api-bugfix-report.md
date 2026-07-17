# EnviToo Backend — Bug Investigation & Fix Report

**Scope:** Full API endpoint verification, error-path audit, and targeted fixes  
**Stack:** Go / Echo / pgx v5 / Redis / Clerk  
**Result:** Initial E2E **67/74**, after fixes **74/74**; later authz/error hardening applied on top  

---

## 1. How the investigation was done

1. **Mapped every route** from `Server.Register` (health, public discovery, guest OTP/registration, auth registration, staff moderation, volunteer public/apply/manage).
2. **Ran unit tests** (`go test ./internal/...`).
3. **Stood up the real server** against Supabase Postgres + local Redis, minting Clerk session JWTs for two existing users (organizer + attendee).
4. **Drove a full lifecycle E2E suite** (create event → guest OTP → register → approve/reject/status → check-in → volunteer roles/apply/approve → cancel).
5. **Traced failures** via HTTP status, response body, and server logs (`SQLSTATE 23505`, cache hits, JSON field names).
6. **Fixed root causes only** — no drive-by refactors; kept existing API contracts and docs semantics.

---

## 2. Bugs found and how they were fixed

### Bug 1 — Unique violations returned **500** instead of **409** (critical)

**Endpoints affected**

| Method | Path | Expected | Observed |
|--------|------|----------|----------|
| `POST` | `/api/events/:id/register` (duplicate) | 409 | **500** |
| `POST` | `/api/events/:id/volunteer/roles` (duplicate name) | 409 | **500** |
| `POST` | `/api/events/:id/volunteer/apply` (duplicate active app) | 409 | **500** |

**What was wrong**

Handlers already mapped sentinel errors like `ErrConflict`, `ErrVolunteerRoleDuplicate`, `ErrVolunteerDuplicate` → HTTP 409. The service layer used helpers:

- `helper.IsRegistrationDuplicate`
- `helper.IsUniqueViolation(..., "volunteer_roles_event_id_name_key")`
- `helper.IsVolunteerApplicationDuplicate`

Those helpers used:

```go
var pgErr *pgconn.PgError  // from github.com/jackc/pgconn  (legacy)
errors.As(err, &pgErr)
```

The app uses **pgx/v5**, which returns:

```text
*github.com/jackc/pgx/v5/pgconn.PgError
```

That is a **different Go type** than legacy `*github.com/jackc/pgconn.PgError`.  
`errors.As` never matched → unique violations were treated as unknown errors → logged and returned as **500 Internal Server Error**.

Server log evidence:

```text
ERROR ... duplicate key value violates unique constraint "registrations_event_id_email_key" (SQLSTATE 23505)
ERROR ... "volunteer_roles_event_id_name_key"
ERROR ... "idx_volunteer_apps_event_user_active"
```

**How it was fixed**

- Switched import in `internal/helper/db_error.go` to `github.com/jackc/pgx/v5/pgconn`.
- Dropped unused legacy `pgconn` from `go.mod` / `go.sum`.
- Added unit tests in `internal/helper/db_error_test.go` that wrap a real `pgx/v5/pgconn.PgError` and assert:
  - unique violation detection
  - registration duplicate
  - volunteer app duplicate
  - foreign-key violation

**Why this is standard practice**

DB constraint errors should be classified once (helper), turned into domain errors (`ErrConflict`), then mapped at the HTTP edge to **409 Conflict** — never leak as 500.

---

### Bug 2 — **Authorization bypass via cache** on staff registration list (security)

**Endpoint**

`GET /api/events/:id/registrations`

**Expected:** non-staff → **404** (docs: event staff only)  
**Observed:** non-staff → **200** with full registration list (including `access_token`)

**What was wrong**

Handler used cache-aside:

```text
key = GET:/api/events/{eventID}/registrations[?query]
loader = ListByEvent(... actorID ...)   // auth check lived only inside loader
```

Flow:

1. Organizer lists registrations → loader runs, staff check passes, **payload cached under a key that does not include actor ID**.
2. Attendee (non-staff) hits the same URL → **cache hit** → loader (and auth) **never run** → staff-only data returned.

This is a classic “shared cache key in front of per-user authorization” failure.

**How it was fixed**

1. Added `RegistrationService.EnsureEventStaff` (thin wrapper around `RequireEventStaff`).
2. In `ListEventRegistrations`, call **`EnsureEventStaff` before `cache.GetJSON`**.
3. Keep the shared staff cache key (organizer and volunteer admin share the same payload), but only after the caller is proven staff.

**Why this is standard practice**

Authorization must run on **every request**. Cache may store the result of an authorized load; it must not replace the auth gate. Pattern: **auth first, then cache**.

---

### Bug 3 — `GET /api/me` returned **PascalCase** JSON (API contract break)

**Endpoint**

`GET /api/me`

**Expected** (API docs / client convention):

```json
{ "id": "...", "auth_user_id": "...", "email": "...", "avatar_url": "...", ... }
```

**Observed:**

```json
{ "ID": "...", "AuthUserID": "...", "Email": "...", "AvatarUrl": "...", ... }
```

**What was wrong**

```go
return c.JSON(http.StatusOK, u)  // *db.User from sqlc
```

sqlc models have **no `json` tags**. Encoding/json falls back to exported field names → PascalCase.  
All other endpoints use DTOs (`EventResponse`, `RegistrationResponse`, etc.) with snake_case tags.

**How it was fixed**

- Added `internal/dto/user.go` with `UserResponse` + `NewUserResponse`.
- `Me` now returns `dto.NewUserResponse(u)`.

**Why this is standard practice**

Never serialize persistence models directly. Boundary types (DTOs) own the public JSON contract.

---

### Bug 4 — `PATCH /api/registrations/:id/status` was not truly **organizer-only** (authz)

**Endpoint**

`PATCH /api/registrations/:id/status`

**Docs:** organizer only; volunteer admins should get **404** (use approve/reject instead).

**What was wrong**

`UpdateStatus` loaded the row with `GetRegistrationByIDAdmin`, which allows **organizer OR volunteer admin**. Then:

```go
if reg.Status == status {
    return reg  // early success
}
```

So a **volunteer admin** sending the current status got **200** and the full registration body — bypassing the organizer-only SQL update that would have returned no rows.

Mutation paths for volunteer admins still failed at `UpdateRegistrationStatus` (organizer-only SQL) → 404, but the no-op path was a real privilege leak.

**How it was fixed**

After the staff-scoped fetch, explicitly require organizer:

```go
isOrg, err := qtx.IsEventOrganizer(...)
if !isOrg {
    return ErrRegistrationNotFound  // 404, matches docs
}
```

Then allow same-status no-op / transitions / capacity checks.

**Why this is standard practice**

Defense in depth: SQL is the source of truth for writes, but **service-layer authz must match the documented actor model** so no-op and read-before-write paths cannot leak or over-permit.

---

### Bug 5 — Incomplete HTTP mapping for `ErrEventNotFound` on moderation paths

**Endpoints**

- `POST /api/registrations/:id/approve`
- `PATCH /api/registrations/:id/status`

**What was wrong**

Service can return `ErrEventNotFound` during capacity / event lock paths. Handlers only mapped registration-not-found, full, forbidden, invalid status — missing event-not-found → **500**.

**How it was fixed**

Map `service.ErrEventNotFound` → **404** `"event not found"` on both handlers.

---

### Bug 6 — Email not normalized on auth user / volunteer apply (data integrity)

**Endpoints**

- `POST /api/events/:id/register`
- `POST /api/events/:id/volunteer/apply`

**What was wrong**

Guest OTP path always ran `normalizeGuestEmail` (trim + lower).  
Auth registration and volunteer apply used profile/override email **as-is**.

Postgres `UNIQUE (event_id, email)` is case-sensitive. Mixed case could:

- create near-duplicate emails for the same event, or
- make duplicate detection inconsistent between guest and user flows.

**How it was fixed**

Apply the same `normalizeGuestEmail` on:

- user registration create
- volunteer application create

**Why this is standard practice**

Normalize identity fields at the write boundary so uniqueness, OTP keys, and lookups share one canonical form.

---

## 3. Files changed

| File | Change |
|------|--------|
| `internal/helper/db_error.go` | Use `pgx/v5/pgconn` for error classification |
| `internal/helper/db_error_test.go` | **New** — regression tests for constraint mapping |
| `internal/dto/user.go` | **New** — public user JSON DTO |
| `internal/server/server.go` | `Me` returns `UserResponse` |
| `internal/server/registration.go` | Auth-before-cache; event-not-found HTTP maps |
| `internal/service/registration.go` | `EnsureEventStaff`; organizer check on `UpdateStatus`; email normalize |
| `internal/service/volunteer.go` | Email normalize on apply |
| `internal/service/event.go` | Calendar ownership + FK → `ErrInvalidEvent` |
| `internal/server/event.go` | Create maps `ErrInvalidEvent` → 400 |
| `internal/middleware/auth.go` | Guest-link invalidator passes `userID` |
| `go.mod` / `go.sum` | Drop legacy `jackc/pgconn` dependency |

---

## 4. Verification

| Check | Result |
|-------|--------|
| Unit tests `./internal/...` | Pass |
| E2E suite (74 cases) after Bugs 1–3 | **74/74** |
| Smoke after Bugs 4–6 | `/api/me` snake_case; create/cancel event OK |
| Duplicate register / role / apply | **409** (post–Bug 1) |
| Non-staff list event registrations | **404** (post–Bug 2) |

---

## 5. What was *not* treated as a product bug

| Observation | Why left alone |
|-------------|----------------|
| Missing `Authorization` → **403** empty body (Clerk middleware), docs say 401 | Correct external middleware behavior; auth still enforced |
| One active volunteer application per user per event | Intended unique index; second apply correctly **409** after Bug 1 |
| Session JWT ~60s lifetime from Clerk Backend API | Test harness concern only |

---

## 6. Summary

| Severity | Count | Themes |
|----------|-------|--------|
| Critical (wrong status / data leak) | 2 | Unique→500; cache auth bypass |
| High (authz / API contract) | 2 | Status PATCH volunteer admin no-op; `/me` PascalCase |
| Medium (correctness / hygiene) | 7+ | Event mapping; email; check-in; calendar; guest-link cache; verified email; volunteer post-commit; event response 500 |

Root themes:

1. **Wrong error type for pgx/v5** broke domain error classification.
2. **Cache placed before authorization** broke staff-only access control.
3. **Persistence models leaked to HTTP** broke the public JSON contract.
4. **Service authz incomplete** on an “organizer-only” path.
5. **Inconsistent normalization** risked uniqueness/identity bugs.

All fixes stay on the main path: correct HTTP semantics, preserve documented actor roles, and keep existing route/DTO shapes (except making `/me` match the rest of the API).

---

### Bug 7 — Check-in miss path returned unstructured **500** (edge case)

**Endpoints**

- `POST /api/registrations/:id/check-in`
- `POST /api/events/:id/check-in`

**What was wrong**

When the UPDATE returned `ErrNoRows` but the registration was approved, not yet checked in, and the event was published, the service fell through to:

```go
return fmt.Errorf("check-in failed: %w", err) // bare pgx.ErrNoRows → HTTP 500
```

That path also **ignored event lookup failures** (`eventErr != nil` was treated as “event is fine”). Concurrent check-in races could likewise surface as 500 instead of an idempotent 200.

**How it was fixed**

- Map missing event → `ErrEventNotFound`.
- Propagate real event fetch errors instead of swallowing them.
- On remaining miss after approved+published, **re-fetch** once; if `checked_in_at` is set, return success (idempotent race win); else `ErrRegistrationNotFound`.
- Handlers map `ErrEventNotFound` → **404**.

---

### Bug 8 — Whitespace-only email override could empty `NOT NULL` email

**Endpoints**

- `POST /api/events/:id/register`
- `POST /api/events/:id/volunteer/apply`

**What was wrong**

After introducing email normalization, an override of `"   "` became `""`, which violates `email TEXT NOT NULL` and produced a raw DB error (**500**).

**How it was fixed**

Normalize profile email first; only apply override when the normalized override is non-empty.

---

### Bug 9 — Event create accepted foreign/invalid `calendar_id` (authz + error mapping)

**Endpoint**

`POST /api/events`

**What was wrong**

1. Any existing calendar UUID could be attached without checking ownership (calendars are 1:1 with users).
2. Non-existent `calendar_id` hit `events_calendar_id_fkey` and returned **500**.
3. `CreateEvent` handler mapped all service errors to 500 — no `ErrInvalidEvent` path.

**How it was fixed**

- Verify `calendar_id` matches the organizer’s calendar via `GetCalendarByUserID`; mismatch / missing → `ErrInvalidEvent`.
- Map FK violations on insert → `ErrInvalidEvent` (`invalid calendar_id`).
- Handler maps `ErrInvalidEvent` → **400**.
- `resolveName` trims whitespace overrides so blank names fall back cleanly.

---

### Bug 10 — Guest-link cache invalidation skipped `GET /api/me/registrations`

**Endpoints / path**

- Auth `LoadUser` guest linking → `GET /api/me/registrations`

**What was wrong**

When guest rows were linked (or superseded guests cancelled) on login, `forgetRegistrationsByEmail` cleared guest-token and event caches but **not** `GET:/api/me/registrations:user:{userID}`.

A returning user with a warm me/registrations cache could miss newly linked guest tickets until TTL expiry.

**How it was fixed**

- Extend `GuestRegistrationInvalidator` to pass `userID`.
- After email-scoped invalidation, also delete `me/registrations` keys for that user.

---

### Bug 11 — Onboarding accepted **unverified** Clerk emails

**Path**

`LoadUser` first-login → `primaryEmail`

**What was wrong**

Error message claimed “no verified email address”, but `primaryEmail` returned the primary/first address **without** checking `Verification.Status == "verified"`. Unverified emails could be stored and used for guest linking.

**How it was fixed**

Only return emails whose verification status is `verified` (prefer primary when verified).

---

### Bug 12 — Volunteer mutate success path could still return **500**

**Endpoints**

- `POST /api/events/:id/volunteer/apply`
- `POST /api/volunteer-applications/:id/approve|reject`
- `DELETE /api/volunteer-applications/:id`

**What was wrong**

After a successful commit/update, handlers re-fetched the application with role. A post-mutation read failure returned **500** even though the write succeeded.

**How it was fixed**

- Build responses from rows already loaded (approve/apply).
- For cancel/reject, degrade role metadata on lookup failure instead of failing the request.

---

### Bug 13 — Event cancel left related caches warm

**Endpoint**

`DELETE /api/events/:id`

**What was wrong**

Only event/slug/public-list keys were deleted. Staff registration list and volunteer settings/roles caches could serve pre-cancel payloads until TTL.

**How it was fixed**

Also invalidate event registrations and volunteer settings/roles keys for that event.

---

### Bug 14 — Successful event mutations could still return **500**

**Endpoints**

- `POST /api/events`
- `PATCH /api/events/:id`
- `DELETE /api/events/:id`

**What was wrong**

After a successful create/update/cancel, handlers called `EventResponseWithCount`. If the count/visibility enrichment query failed, the client got **500** even though the write committed — clients often retry and create **duplicate** events.

**How it was fixed**

- Added `EventResponseBestEffort`: prefer full enrichment; on failure return the core event (count 0, best-effort visibility) and log a warning.
- Create/update/cancel always return the mutation result with the appropriate success status.

---

### Bug 15 — Event list enrichment failure failed the whole list

**Endpoints**

- `GET /api/events`
- `GET /api/me/events`

**What was wrong**

After the underlying event rows were loaded successfully, a failure in registration-count or visibility enrichment returned **500** for the entire list, even though the primary data was available.

**How it was fixed**

- Added `EventResponsesBestEffort` (batched enrichment with partial fallback).
- Public list and my-events use best-effort enrichment so degraded secondary queries do not hide the list.

---

## 7. Recommended follow-ups

1. Commit this set with a clear message.
2. Cancel the recurring scheduler job `019f70af1449` if it is still active.
3. Add a small integration test that asserts “cache hit still returns 404 for non-staff” so Bug 2 cannot regress.
4. Add unit tests for check-in race re-fetch and blank email override fallback.
