# EnviToo — Technical Specification

> Generated from a full read-through of the repository. Source of truth: the code itself
> (this doc is derived, so when code and doc disagree, the code wins). File paths and
> line numbers are cited where stable.

## 1. System Overview

**EnviToo** is a community event management platform:

- **Discover & register** for public events (search, category filters, virtual/physical)
- **Host events** with attendee management, registration approval, QR check-in, and an on-site admin dashboard
- **Guest registration** via email OTP (no account required) with magic-token tickets
- **Volunteer programme**: organizers define roles + settings → users apply → organize approval → roster
- **Email pipeline**: transactional outbox → NATS JetStream → Resend batch API
- Payments: `price_in_paise` is stored on events and a `payments` (Razorpay-shaped) table exists in migrations, but **no payment processing is implemented** in app code.

| Layer | Tech | Entry point |
|---|---|---|
| API | Go 1.26.4, Echo v5, pgx/v5 + sqlc, Clerk SDK v2.7.0 | `backend/cmd/server` (:8080) |
| SPA | React 19, Vite 8, React Router v7, TanStack Query v5, Zustand, zod, Tailwind v4, Clerk React | `client/` (:5173) |
| Infra | PostgreSQL 17, Redis 7, NATS JetStream 2.10, Resend, Docker Compose | root `docker-compose.yml` |
| CI/CD | GitHub Actions: backend/frontend checks + gitleaks + docker image publish on `deploy` branch | `.github/workflows/` |

Three Go binaries: `server` (HTTP API), `worker` (outbox drainer → NATS), `mailer` (NATS → Resend batch).

Repo layout:

```
backend/
  cmd/server/       server binary (HTTP API :8080)
  cmd/worker/       outbox drainer
  cmd/mailer/       NATS → Resend batch sender
  internal/
    server/         Echo wiring + HTTP handlers (server.go, + handler files)
    service/        domain logic (event.go, registration.go, volunteer.go, guest_otp.go, event_access.go, storage.go, errors.go)
    middleware/     auth.go, clerk.go, loaduser.go, ratelimit.go, idempotency.go, logger.go
    dto/            validation + request/response types
    db/             sqlc generated code
    cache/          redis cache store
    email/          outbox drainer, nats config, batcher, sender, templates
    internal/database/  pool open/ping/close (database.go)
  sql/              sqlc query sources (*.sql)
  supabase/migrations/   all DDL (32 files; app doesn't use ORM)
  /client/src/     SPA (lib/api, hooks, pages, components/domain, components/ui)
Caddyfile          reverse proxy config (prod)
docker-compose.yml / docker-compose.prod.yml
Dockerfile         multi-stage build (server/worker/mailer binaries; see README §docker)
Makefile, observability-plan.md (planning artifact)
```

## 2. Backend

### 2.1 Request pipeline & global middleware

`Server.Register` (`backend/internal/server/server.go:80`) wires global middleware in order:

1. `IPExtractor = ExtractIPFromXFFHeader(TrustLoopback(true))`
2. `echoMiddleware.Recover()` → `RequestID()` → `RequestLogger` (order matters in Echo: formerly `e.IPExtractor`, then `Use(Recover)`, `Use(RequestID)`, `Use(RequestLogger)`)
3. `BodyLimit("1MB")` → `ContextTimeout(30s)` → CORS (`AllowOrigins: ["*"]`, exposes `Idempotent-Replayed`)

**Route groups** and their middleware chains:

| Route | Method | Auth | Rate limiter | Idempotency |
|---|---|---|---|---|
| `/healthz` | GET | none | none | — |
| `/e/:slug` (social prerender) | GET | none | public (10 rps/20 burst) | — |
| `/api/events`, `/api/events/:id`, `/api/events/slug/:slug` | GET | none | public | — |
| `/api/events/:id/register/guest/otp` | POST | none | OTP (5/min) | ✅ |
| `/api/events/:id/register/guest/otp/verify` | POST | none | OTP | — |
| `/api/events/:id/register/guest` | POST | none | public | ✅ |
| `/api/guest-registrations` | GET/DELETE | none | public | — |
| `/api/events/:id/volunteer/settings` /`/roles` | GET | public | public | — |
| `/api/me` | POST | Auth only | auth (30 rps) | — |
| `/api/me` | GET | protected | auth | — |
| `/api/events` | POST | protected | auth | ✅ |
| `/api/me/events` | GET | protected | auth | — |
| `/api/events/:id` | PATCH / DELETE | protected | auth | — |
| `/api/upload/presigned-url` | POST | protected | auth | — |
| `/api/events/:id/register` | POST | protected | auth | ✅ |
| `/api/registrations/:id` | GET / DELETE | protected | auth | — |
| `/api/me/registrations` | GET | protected | auth | — |
| `/api/events/:id/registrations` | GET | protected | auth | — |
| `/api/events/:id/check-in` | POST | protected | auth | — |
| `/api/registrations/:id/admin` | GET | protected | auth | — |
| `/api/registrations/:id/approve` | POST | protected | auth | ✅ |
| `/api/registrations/:id/reject` | POST | protected | auth | — |
| `/api/registrations/:id/check-in` | POST | protected | auth | — |
| `/api/registrations/:id/status` | PATCH | protected | auth | — |
| `/api/events/:id/volunteer/apply` | POST | protected | auth | ✅ |
| `/api/me/event-staff` / `/me/volunteer-applications` | GET | protected | auth | — |
| `/api/volunteer-applications/:id` | GET/DELETE | protected | auth | — |
| `/api/events/:id/volunteer/settings` | PUT | protected | auth | — |
| `/api/events/:id/volunteer/roles` | POST / `:roleId` DELETE | protected | auth | — |
| `/api/events/:id/volunteer/applications` | GET | protected | auth | — |
| `/api/events/:id/volunteers` | GET | protected | auth | — |
| `/api/volunteer-applications/:id/admin` | GET | protected | auth | — |
| `/api/volunteer-applications/:id/approve` / `reject` | POST | protected | auth | — |

Protected = `middleware.Auth(jwksClient)` → `LoadUser(pool, queries, cache, forgetRegistrationsByEmail)` → `AuthRateLimiter(30 rps)`.

### 2.2 Middleware details

| File | What it does |
|---|---|
| `middleware/auth.go` | Clerk `RequireHeaderAuthorization` verification against a local JWKS key set |
| `middleware/clerk.go` | JWKS caching backend (TTL 50 min), pre-warmed at startup via `InitJWKSClient` — zero outbound HTTP on the hot path |
| `middleware/loaduser.go` | Cache-first `SELECT users WHERE auth_user_id = $1` (`user:auth:<clerk_id>`, TTL 2m). On miss: `FindOrProvision` (upsert in a single CTE) then cache. After resolution runs `linkGuestRegistrations` (matches `registrations WHERE email = $1 AND user_id IS NULL`, links `user_id` and cancels the guest tokens in one CTE) and fires a cache-invalidation callback (`forgetRegistrationsByEmail`) for the guest-registration cache key. Return path: user in context + guest regs linked |
| `middleware/ratelimit.go` | In-process token bucket (per-IP + per-user). `DefaultPublicConfig` 10 rps / burst 20; `DefaultOTPConfig` 5 req/min burst 5 (strict — each OTP request can enqueue outbound email); `DefaultAuthConfig` 30 rps |
| `middleware/idempotency.go` | `Idempotency-Key` header (UUID). Redis Lua claim: key → `in_progress` TTL 60s → on success `done` TTL 5m. On replay, returns the cached body + `Idempotent-Replayed: true` header |
| `middleware/logger.go` | `RequestLogger` (structured slog; ignores GET /healthz) |

### 2.3 Services

**`EventService`** (`backend/internal/service/event.go`):
- `Create` — validates (title/description/URL/capacity etc.), generates slug (retry loop up to `maxEventSlugAttempts=10`, `ON CONFLICT (slug) DO NOTHING` → re-roll on `pgx.ErrNoRows`), upserts `event_visibility`, default `published`.
- `GetViewableByID` / `GetEventBySlug` — resolves status (published/cancelled/completed; draft/deleted → 404).
- `ListPublic` — `?limit` default 20 clamp 100; `?offset`; `?q` (title/description ILIKE), `?category`, `?virtual` (0=all,1=virtual,2=physical), time range `starts_from`/`starts_to`, order by `starts_at ASC`. Enriches `registration_count` per event.
- `Update` — role-aware resolution; organizer vs volunteer-admin edits (admin can only patch `description`, `cover_image_url`, `venue_name`, `location_address`, `meeting_url`); capacity may not go below current active registration count; cross-field validation on merge; `event_visibility` upsert.
- `Cancel` — soft-cancel (status `cancelled`), keeps registrations/volunteers; outbox emails to organizer + all active registrants (dedup by email).

**`RegistrationService`** — creating, updating, cancelling, approving, check-in.
- `CreateGuest` — requires OTP-verified Redis marker `guest:verified:{event_id}:{email}`; status `pending` if `requires_approval` else `approved`; guest regenerates access_token; unique per event+email (one active registration enforced by partial unique index); outbox `guest_registered`.
- `Create` — host flow: email always derived from authenticated account (`name` optional, defaults from profile); `FOR UPDATE` lock on event row inside tx prevents over-capacity inserts; `ErrEventFull` on full.
- `CancelGuest` / `Cancel` — soft-cancel by id + user_id/event match; outbox `guest_cancelled`.
- Admin approve/reject/status — organizer or volunteer-admin gated; approve re-checks capacity under lock; status transitions whitelist.
- `CheckIn` (by registration id, `FOR UPDATE`) / `CheckInByEventToken` (by access_token + event id) — idempotent; outbox `checked_in` mail.
- Cap-gate helper `validateCapacityForApproval`.

**`VolunteerService`** — `volunteer.go`:
- `UpsertSettings` / `CreateRole` / `DeleteRole` (FK-violation → `ErrVolunteerRoleInUse`; `volunteer_roles_event_id_name_key` → `ErrVolunteerRoleDuplicate`)
- `Apply` — locks event + checks `settings.enabled`, `max_applications` count, role belongs to event, duplicate → `ErrVolunteerDuplicate` + outbox `application_confirm`
- `ListStaffEventsByUser` — joins events/apps/roles
- `Approve` — Roster insert `ON CONFLICT (event_id,user_id,role) DO NOTHING` idempotent + outbox `volunteer_status_change`
- `Reject`, differentiate `volunteerModerationError` (403 admin-role restriction via `ErrVolunteerAdminRoleRestricted`)

**`GuestOTPService`** — `guest_otp.go`:
- Constants: `guestOTPTTL=10m`, `guestVerifiedTTL=15m`, `guestOTPCooldown=60s`, `guestOTPMaxTries=5`, `guestOTPCodeBound=1_000_000`
- HMAC-hashed 6-digit code, per-event+email cooldown via Redis marker, 503 when `OTP_SECRET` unset (optional — outbox email send is best-effort)
- Flow: `RequestGuestOTP` (cooldown + store) → `VerifyGuestOTP` (hash match + attempts + set `guest:verified:`) → `POST /register/guest` (consumes verified marker)

**`EventAccess`** — `event_access.go`: `EventStaffRole` iota (`None=0, Organizer=1, VolunteerAdmin=2`), `ResolveStaffRole` (organizer first, then approved admin application), `ErrEventStaffForbidden` guard for staff-scoped handlers.

**`StorageService`** — `storage.go`:
- S3-compatible presigned URL generation (R2). Route: `POST /api/upload/presigned-url` → returns `{url, key, method:"PUT"}`; client uploads direct to R2, then binds the key intto `cover_image_url` on event create/update.

### 2.4 DTO types

- `CreateEventRequest`: `title (required, 1-200), description (omitempty,≤5000), starts_at/ends_at (required, ends>starts), capacity (required,≥1), price_in_paise (≥0), currency (len=3), requires_approval (bool), cover_image_url (omitempty,url), calendar_id (uuid), is_virtual (bool), meeting_url (url, required when virtual), venue_name (≤200), location_address (≤500), lat/lng (numeric), category ∈ {technical, cultural, sports, workshop, seminar, hackathon, other}, visibility ∈ {public, unlisted}`
- `UpdateEventRequest` — all optional pointers (incl. `**string cover_image_url` to distinguish "clear → null" vs omitted)
- `CreateRegistrationRequest` — `name` optional (1–200); email always from auth
- `CreateGuestRegistrationRequest` — `name` required (1–200), `email` required (email format)
- `CheckInRequest` — `access_token` required (UUID)
- `UpdateRegistrationStatusRequest` — `status required ∈ {pending, approved, rejected, cancelled}`
- Volunteer: `UpsertVolunteerSettingsRequest` (`enabled bool, description ≤2000, max_applications ≥1`), `CreateVolunteerRoleRequest` (`name 1–100, is_admin bool`), `CreateVolunteerApplicationRequest` (`role_id required, name 1–200, email optional+format, phone ≤20, description 1–5000 required`)
- `PresignedURLRequest` — `filename ≤255`, `content_type ∈ {image/jpeg, image/png, image/webp}`, `size ∈ [1B, 5MB]`
- Responses: `EventResponse`, `RegistrationResponse`, user response — exact field names below.

**Response shapes** (all list endpoints wrap `{items, limit, offset, total}`):

```
EventResponse {
  id, organizer_id(omitempty), title, description*, starts_at,
  ends_at, capacity, registration_count, price_in_paise, currency,
  status, visibility, slug, category*, requires_approval,
  cover_image_url*, calendar_id*, is_virtual, meeting_url*,
  venue_name*, location_address*, lat*, lng*, warning*,
  created_at, updated_at          (* = omitempty)
}
RegistrationResponse {
  id, event_id, user_id(omitempty), name, email, status,
  access_token(omitempty, set only on create), token_expires_at*(omitempty, create only),
  checked_in_at*, event(omitempty — event object in admin/list views), created_at, updated_at
}
UserResponse { id, email, username, avatar_url, created_at }
TokenResponse { token }   // guest OTP verify response
UploadPresignedResponse { url, method: "PUT", key, fields }
```

### 2.5 SQL layer (`backend/sql/*.sql` — sqlc)

- `event.sql` — `CreateEvent` (returning id+slug with ON CONFLICT DO NOTHING retry), `GetEventByID/Fill`, `GetEventBySlug`, `GetEventByIDForUpdate` (FOR UPDATE lock for admissions), `EventBySlug`, `CopyEvent` (duplicate event), `ListEvents`, `CountEvents`, `ListPublicEvents` (filters + pagination + counts), `RandomSlug` variants, `Insert/Update/Delete` + visibility upsert
- `visibility.sql`, `event_visibility` (PK event_id, `visibility ∈ public|unlisted`)
- `registration.sql` — `CreateRegistration` (+ guest with `ON CONFLICT` on `uq_access_token`), `GetRegistrationByID/ByAccessToken`, `CountRegistrationsByEvent`, `RegisterForEvent`, `LinkGuestRegistrations`, `CancelRegistration`, `ApprovedCount`, `ListRegistrationsByEvent/User`, `IsOnEventRoster`… (query names per current file — verify exact list when editing)
- `volunteer.sql` — `UpsertVolunteerSettings`, `GetVolunteerSettingsByEvent`, `GetVolunteerRoleByID`, `DeleteVolunteerRole`, `ListVolunteerRolesByEvent`, `CreateVolunteerApplication`, `ListMyVolunteerApplications`, `CancelVolunteerApplication`, `GetVolunteerApplicationByIDAdmin`, `ListEventVolunteerApplications` + `ListEventVolunteers` + `ApproveVolunteerApplication`/`RejectVolunteerApplication` (SQL-embedded authz: organizer or `is_admin=false` role + admin applicant), `InsertVolunteer` (ON CONFLICT DO NOTHING), `CountVolunteerApplicationsByEvent`
- `user.sql` — `FindOrProvision` (CTE upsert), `GetDefaultUserByClerkID`, `LinkGuest`
- `email_outbox.sql` — INSERT used inside every write tx

### 2.6 Config & boot

- `internal/config/config.go` — env allow-list; `Load()` reads `DB_URL` (required, fail-fast), `CLERK_SECRET_KEY`, `REDIS_URL`, `OTP_SECRET`, `HEALTHZ_TOKEN`, `R2_*` (bucket/endpoint/access keys). `LoadWorker` reads `WORKER_` prefix (worker NatsURL + poll interval).
- `cmd/server/main.go` — parse config → pgx pool → sqlc `db.New(pool)` → Redis cache → Clerk JWKS prewarm → services → echo instance → `server.Register(ctx, e)` → run on `:8080` with graceful shutdown (SIGINT/SIGTERM).
- `cmd/worker/main.go` — Drainer loop (polls `email_outbox` every 2s, batch 50, `FOR UPDATE … SKIP LOCKED`), publishes to NATS `EMAILS` stream messages with MsgID = outbox UUID.
- `cmd/mailer/main.go` — Batcher loop: `Fetch(NoWait/500ms)` up to 100 msgs → render jinja + Resend `Batch.SendWithOptions` (permissive validation) → per-index Ack/Nak(2s) → Term on unparseable payload. Rate-limited to 4 req/sec (leaves 1 req/sec headroom vs Resend 5/sec team limit).

### 2.7 Testing

- `backend/.github` — workflows in this repo are **client-side only** (see "CI"). Backend tests: `go test ./...` with `TEST_DB_URL` (Supabase alternate). `cmd/mailer` has a unit test harness for render+batch.

## 3. Frontend

### 3.1 Architecture

- React 19 + Vite 8 + TS (strict), `@/` → `src`.
- React Router v7 (`createBrowserRouter`, lazy-loaded routes, `SiteLayout` with Header/Footer/BottomNav).
- TanStack Query v5 for server state; Zustand for auth state (`useAuthStore.js` — actually `client/src/lib/auth-store.ts`; verify naming).
- Clerk React (`<ClerkProvider publishableKey=...>`) gate in `providers.tsx`; auth token via `useAuth().getToken()`, injected by `api` client chain.
- App shell: `main.tsx` → `providers.tsx` → `App.tsx`. `ProtectedRoute` wraps user-only routes; `PublicOnlyRoute` for `/signin`, `/signup`.

Client API layer (`src/lib/api/`):

| File | Role |
|---|---|
| `index.ts` | `createApi({ baseUrl, getToken })` — fetch wrapper + `ApiError` + zod parse |
| `client.ts` | Api class ~ 38 methods, one per endpoint |
| `endpoints.ts` | route/URL build conventions |
| `schemas.ts` | zod schemas mirrored from backend DTOs |

**Query key conventions** (TanStack):
- events: `["events", id]`, `["events","slug",slug]`, `["events","public",params]`, `["events",{mine,list,limit,skip}]`
- registrations: `["registrations", id]`, `["registrations","event",eventId]`, `["registrations","mine",...]`
- volunteers: `["volunteer","settings",eventId]`, `["event-staff","mine"]`, `["volunteer-applications","mine"]`
- mutations invalidate the **smallest** matching prefix (never blanket `queryClient.clear`), e.g. cancel event → `["events"]`, check-in → registration+event keys.

**State/auth**: `use-account-setup.ts` calls `POST /api/me` on first load (provisioning), then `GET /api/me` for the profile; `use-onboarding.ts` shows `OnboardingChecklist` gate on `/welcome`.

### 3.2 Routing table (`src/App.tsx`)

| Path | Page | Notes |
|---|---|---|
| `/` | Landing | |
| `/discover` | Discover | filters + virtual toggle + category pills |
| `/e/:slug` | EventDetail | event view + register CTA |
| `/event/:id` | EventDetail (alias) | |
| `/ticket/:token` | GuestTicket | magic token — guest ticket page submits to /api/guest-registrations |
| `/privacy` `/terms` | Legal | |
| `/welcome` | Welcome + OnboardingChecklist | Protected |
| `/home` | MyTickets | Protected |
| `/create` | CreateEvent | Protected |
| `/manage` | ManageEvents | Protected |
| `/manage/:id` | EventDashboard (tabs: Overview / RegistersPanel / Volunteers / CheckInScanner) | Protected |
| `/manage/:id/edit` | EditEvent | Protected |
| `/signin/*` `/signup/*` | SignIn / SignUp | PublicOnlyRoute |
| `*` | NotFound | |

### 3.3 Key components

- `components/domain/RegisterForm.tsx` (guest-vs-account switch, name field)
- `CheckInScanner.tsx` — QR parse (html5-qrcode or jsQR) → `checkInByEventToken` → ding + toast
- `TicketPDF.tsx` — prints ticket as PDF (guest page)
- `EventMap.tsx` + `LocationSearch.tsx` — geocoding via `lib/geocoding.ts`
- `ImageUpload.tsx` — presigned-URL flow for cover images
- `Volunteer*Panel.tsx` — settings/roles/applications/roster with status badges
- `OnboardingChecklist.tsx` — step list gating profile completion

### 3.4 Styling

Tailwind v4 (`@theme` config in `vite.config.ts`); `components/ui/*` adopt shadcn-style primitives (button, input, dialog, card, table, tabs, switching) — consistent with a design system; dark mode via `ThemeToggle`.

## 4. Database Schema (migrations source-of-truth)

**users** (`backend/supabase/migrations/20260625084255_create_users.sql`): `id uuid PK default gen_random_uuid()`, `auth_user_id text UNIQUE NOT NULL` (Clerk subject), `email text UNIQUE NOT NULL`, `username`, `avatar_url`, created/updated. Index `idx_users_auth_user_id`. Note: **no mutable-profile column** (name/username nullable).

**calendars**: `user_id UNIQUE`, `slug UNIQUE` (partner profile pages).

**events**: `id, organizer_id FK users, title, description, location (dropped → venue fields), starts_at, ends_at, capacity CHECK >0, price_in_paise default 0, currency default 'INR', status CHECK(published|cancelled|completed) default published, slug UNIQUE, requires_approval default false, cover_image_url, calendar_id FK, is_virtual + meeting_url + venue_name + lat/lng (NUMERIC(9,6)) with CHECK (virtual ⇒ has meeting_url; physical ⇒ venue+coord), created_at, updated_at`. Indexes on organizer_id, starts_at, status, slug.

**registrations**: `id PK, eventID FK → events ON DELETE CASCADE, userID FK → users ON DELETE SET NULL (guest rows have NULL), name, email, status CHECK(pending|approved|rejected|cancelled), access_token UNIQUE, token_expires_at NULL=never, checked_in_at, created/updated`. Uniqueness: `UNIQUE(round (event_id, email))` originally; **migration 20260729120000 replaces with partial unique indexes** `(event_id, email)` and `(event_id, user_id)` where `status IN ('pending','approved')` — re-registration after cancel/reject allowed.

**payments**: `id PK, registration_id FK, event_id FK, user_id FK SET NULL, order_id UNIQUE, payment_id UNIQUE, signature, amount_in_paise, currency, status CHECK(created|paid|failed|refunded), timestamps`. **No app code writes it yet.**

**volunteer_settings**: `id PK, event_id UNIQUE FK, enabled bool, description, max_applications, timestamps`.

**volunteer_roles**: `id PK, event_id FK, name, is_admin bool, UNIQUE(event_id, name)`.

**volunteer_applications**: `id PK, event_id FK CASCADE, user_id FK CASCADE, role_id FK RESTRICT, name, email, phone, description, status CHECK, partial UNIQUE(event_id, user_id) WHERE status IN(pending, approved)`.

**volunteers** (roster): `event_id FK, user_id FK, role_id FK, ON CONFLICT DO NOTHING` idempotent.

**kanban_* / kanban_columns, kanban_tasks, kanban_task_status_log**: **tables exist for a board feature; no API routes or UI surface yet** (see Gaps).

**email_outbox**: `id UUID PK, recipient, template, template_data JSONB, status ENUM(PENDING|PUBLISHED), retries, last_error, created_at, published_at` + partial index on pending.

## 5. Business Flows

### 5.1 Guest registration (public)

1. Frontend collects guest name + email → `POST /api/events/:id/register/guest/otp` → backend hashes to 6 digits, writes Redis (TTL 10m, 60s cooldown per (event,email)), outbox INSERT `guest_otp` in same tx → worker drains → mailer sends.
2. Guest enters code → `POST …/otp/verify` → backend compares HMAC, increments tries (max 5), on success sets Redis `guest:verified:{event}::{email}` (15m).
3. `POST …/register/guest` (idempotency-key) → reads verified marker + kills it (one-time), creates registration row (status pending/approved per `requires_approval`), issues `access_token` UUID (token 30d, returned only on this response), outbox guest ticket email.
4. Ticket page `/ticket/:token` renders QR (PDF download) with check-in.

### 5.2 Host registration + approve/waitlist

`POST /api/events/:id/register` requires Clerk auth; email always derived from the account; `requires_approval` → `pending`, else `approved` (auto). Organizer rejects/approves from `EventDashboard` → outbox status-change email; rejects free the capacity. Check-in via registration list (`POST /api/registrations/:id/check-in`) or QR-token flow.

### 5.3 check-in

EventDashboard's CheckInScanner or admin list: resolve token (GET registration by event `GetRegistrationByAccessToken`) → idempotent update `checked_in_at` (dedupe) → outbox `checked_in` email fires only on the first check-in, inside the same tx.

### 5.4 Email pipeline

(Three-process transactional outbox — already documented in AGENTS.md §"Email pipeline".)

### 5.5 Foreign caching

`internal/cache/store.go` — nil-safe Redis wrapper; keys `GET:/api/events/{id}`, `GET:/api/events/slug/{slug}`, `GET:/api/events?...`; invalidation `forget` hooks on registration/volunteer writes; `GET /api/me` uses LoadUser’s `user:auth:` key (TTL 2m).

## 6. Security model

- Clerk JWT verification via local JWKS (pre-warmed); `CLERK_SECRET_KEY` required for auth routes.
- OTP: HMAC-SHA256 of code+secret (never stored plaintext); 5 retries, cooldown, TTL.
- Access tokens: random UUID, returned once at creation, then only used for token-thread check-in + guest lookup; readable via `/api/guest-registrations?token=`.
- Idempotency-Key prevents duplicate charges/reg; CORS restricted to `*` on API (public API) but no credentials.
- Rate limits (public 10rps/20burst, OTP 5/min, auth 30rps) + per-IP extraction from XFF with TrustLoopback.
- Message size cap 1MB, timeouts 30s, DB pool limits.

## 7. What's still missing / unfinished

### Database-backed but unwired

- **Payments**: `payments` table + `price_in_paise`/`currency` on events exists, but there is no checkout API, no Razorpay integration, no payment status flow. Registration currently ignores price.
- **Kanban / project board**: tables (`kanban_columns`, `kanban_tasks`, `kanban_task_status_log`, migration `20260708230420_kanban_revamp`) exist — tthe server and client have zero code for it.

### Frontend-only gaps

- **Calendar / partner profile pages**: `calendars` table + slug exist; no public calendar page route, no API endpoints for calendars beyond create-on-event.
- **Search for hosts** — none implemented (no user/profile search API); hosts are only discoverable via events.
- **Payment/checkout UI** (obviously follows the API gap).
- **Waitlist**: no waitlist concept — full events simply reject new registrations (`ErrEventFull`).

### Product/UX

- **Capacity shrink UX**: backend guards shrinking capacity below active registration count (`ErrInvalidEvent`), but the EditEvent page surfaces the server error only.
- **Notifications for organizers** (email only; no in-app).
- **Reject reason input** / custom email bodies.
- **Shared `unlisted` events detail** — currently detail works for guests via slug.

### Platform

- **No auth-zero / magic link for guests**; guest flow posts OTP only.
- **No global admin role** — organizers are event-bound; no global admin (reports, bans).
- **No scheduler** — no job that flips `started`/`completed` statuses or auto-cancels stale events.
- **No rate limits on OTP from same email across events** (per-event+email only).

## 8. Conventions & invariants

1. **Single source of truth for DTOs is the Go backend** (`internal/dto/`); the client `schemas.ts` mirrors it manually — re-sync when adding fields.
2. **Transactions write the outbox row in the same TX** (at-least-once email).
3. All state-changing mutations use `Idempotency-Key` where heavy (create/register/check-in/approve).
4. Never return raw `access_token` from list endpoints; only from creation.
5. Rate limits exist at every layered boundary.
6. Cache keys are exact URLs — changing a cache key format or TTL requires updating `internal/cache/store.go` + dependent invalidators.
7. sqlc query changes require regeneration (`cd backend && go generate ./...` or sqlc generate) and will fail CI if the `.sql` files and generated code drift.