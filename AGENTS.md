# AGENTS.md

## Cursor Cloud specific instructions

EnviToo is an event management platform with a **Go API** (`backend/`, port 8080) and **React/Vite SPA** (`client/`, port 5173). See the root `Makefile` for standard commands.

> **Full reference`: `docs/TECHNICAL_SPEC.md` — complete API route table, DB schema, business flows, and gap list. AGENTS.md is the quick-reference; the spec is the deep dive.

### System dependencies (one-time on a fresh VM)

These are **not** in the startup update script:

- **Go 1.26+** at `/usr/local/go` (the repo's `go.mod` requires 1.26.4; system `/usr/bin/go` may be older)
- **Docker** with Compose v2 plugin

### Environment files (required to run the app)

Copy templates and fill in secrets (never commit `.env` files):

| File | Required vars |
|------|----------------|
| `backend/.env` | `DB_URL` (remote Supabase for prod), `CLERK_SECRET_KEY`; optional `REDIS_URL`, `OTP_SECRET`, `TEST_DB_URL` |
| `backend/.env.local` | Overrides `DB_URL`, `REDIS_URL`, `NATS_URL` to point at local docker services (provided; gitignored) |
| `client/.env` | `VITE_CLERK_PUBLISHABLE_KEY` |

Clerk keys come from the Clerk dashboard or `npx clerk@1.5.0 env pull` after `clerk auth login` + `clerk link`. **Valid Clerk keys are required** for the frontend to load public event data (the API client uses `useAuth()` from Clerk even for unauthenticated routes).

### Starting services

```bash
export PATH="/usr/local/go/bin:$PATH"
make dev   # docker infra (postgres+redis+nats) + local backend:8080 + frontend:5173
```

- `make dc-up` — infra only (postgres, redis, nats via docker-compose)
- `make dc-full` — full stack in docker (infra + backend container)
- `make dc-down` — stop all docker services

Migrations run automatically on first `dc-up` (mounted as init scripts). To re-run: `docker compose down -v && make dc-up`.

### Email pipeline (three-process architecture)

```
┌─────────────┐     INSERT email_outbox (in TX)     ┌───────────────┐
│  API Server  │ ──────────────────────────────────► │  PostgreSQL    │
│ (cmd/server) │                                      │  email_outbox  │
└─────────────┘                                      └──────┬────────┘
                                                            │
                                          SELECT FOR UPDATE SKIP LOCKED
                                          (worker polls every 2s, batch 50)
                                                            │
                                                    ┌───────┴───────┐
                                                    │  Outbox Drainer │
                                                    │  (cmd/worker)   │
                                                    └───────┬───────┘
                                                            │
                                          PublishAsync(MsgID=outbox UUID)
                                          subject=emails.<topic>
                                                            │
                                                    ┌───────┴───────┐
                                                    │  NATS JetStream │
                                                    │  Stream: EMAILS │
                                                    │  Consumer:       │
                                                    │  email-resend    │
                                                    └───────┬───────┘
                                                            │
                                          Batcher pulls up to 100 msgs
                                          every 500ms, rate-limited
                                          to 4 req/sec
                                                            │
                                                    ┌───────┴───────┐
                                                    │  Mailer Batcher │
                                                    │  (cmd/mailer)   │
                                                    │  Render +       │
                                                    │  SendBatch (100)│
                                                    └───────┬───────┘
                                                            │
                                              Resend batch API
                                              (1 call = up to 100 emails)
                                                            │
                                                    ┌───────┴───────┐
                                                    │  User Inbox     │
                                                    └────────────────┘
```

**Key design decisions:**

- **Transactional outbox**: email rows are inserted in the same DB transaction as the domain write. If the transaction rolls back, the email is never sent. Guarantees at-least-once delivery.
- **Batch sending**: the mailer uses `client.Batch.SendWithOptions()` (permissive validation) to send up to 100 emails per Resend API call. Each batch call counts as 1 request against Resend's 5 req/sec team limit. The batcher is rate-limited to 4 req/sec via `golang.org/x/time/rate`, leaving 1 req/sec margin.
- **Pull consumer**: the mailer uses `consumer.Fetch(100, FetchMaxWait(500ms))` instead of the old push-based `Consume()` callback. This gives the batcher explicit control over when and how many messages to pull.
- **Single instance**: at 4 batch calls/sec × 100 emails/batch = 400 emails/sec theoretical capacity, one mailer instance handles the full peak load (~85 emails/sec). No horizontal scaling needed.
- **Partial failure handling**: on `BatchEmailResponse.Errors`, successful messages are Ack'd and failed messages (by index) are Nak'd with a 2s delay for redelivery. Full batch failures Nak all.
- **Poison messages**: unparseable JSON payloads are Term'd (removed permanently from the stream).
- **Idempotency**: the batch idempotency key is set to the first outbox UUID in the batch.

**Files:**

| File | Role |
|------|------|
| `internal/email/batcher.go` | Pull-based batch processor (Batcher.Run loop) |
| `internal/email/sender.go` | BatchSender interface + SendBatch using Resend batch API |
| `internal/email/nats.go` | Stream/consumer config (MaxAckPending=200, MaxRequestBatch=100) |
| `internal/email/outbox.go` | Drainer: polls email_outbox, publishes to NATS |
| `cmd/mailer/main.go` | Entry point: wires Batcher → Run until shutdown |
| `cmd/worker/main.go` | Entry point: wires Drainer → PublishBatch loop |

### Optional: running the email pipeline locally

```bash
make worker  # outbox drainer (polls DB → publishes to NATS)
make mailer  # batch sender (pulls NATS → renders → Resend batch)
```

Both need NATS + Resend configured. The mailer also requires `APP_PUBLIC_URL` for ticket links in rendered emails.

### Lint / test / build

| Target | Command |
|--------|---------|
| All checks (CI / pre-commit) | `make check` (alias: `make ci`) |
| Backend tests | `make test` (or `cd backend && go test ./...`) |
| Frontend lint | `cd client && npm run lint` |
| Frontend typecheck | `cd client && npm run typecheck` |
| Production build | `make build` |

Pre-commit hooks (via `make setup`): runs `make check` plus staged gitleaks scan, binary check, and `.env` guard. Gitleaks is optional locally if not installed; GitHub Actions runs gitleaks on every push and pull request (see `.github/workflows/ci.yml`).

### Architecture: `GET /api/me` hot path

`GET /api/me` goes through three middleware layers before the handler:

1. **`Auth()`** — Clerk JWT verification (local JWKS, pre-warmed at server startup via `InitJWKSClient`; zero outbound HTTP)
2. **`LoadUser()`** — User provisioning + guest registration linking (see below)
3. **`AuthRateLimiter`** — Per-user rate limit

The handler itself is a trivial context read + JSON serialize.

**Optimized `LoadUser` flow:**

```
Request → Cache lookup (key: "user:auth:<clerk_user_id>")
  ├─ Cache HIT → user found, skip DB
  └─ Cache MISS → SELECT FROM users WHERE auth_user_id = $1
       ├─ Found → cache in Redis (TTL 2m), continue
       └─ Not found → cold path (Clerk API + onboarding transaction)
```

After user resolution, `linkGuestRegistrations` runs (only on cache miss / DB load):

```
SELECT EXISTS (1 FROM registrations WHERE email=$1 AND user_id IS NULL)
  ├─ False → skip writes (>99% of requests)
  └─ True → single CTE (combines link + cancel in one round-trip)
       └─ If rows affected → fire cache invalidation callback
```

**Result for a returning user with no orphaned guest registrations:**
- Cache hit: **0 DB round-trips** before the handler
- Cache miss: **1 DB round-trip** (SELECT) — no UPDATEs

### Cache conventions

Redis cache keys follow `METHOD:path` patterns (e.g. `GET:/api/events/123`). All reside in `internal/cache/store.go`:

| Key pattern | TTL | Invalidated by |
|-------------|-----|---------------|
| `user:auth:<clerk_id>` | 2m | No mutation endpoint exists yet — add invalidation when creating PATCH /api/me |
| `GET:/api/events/<id>` | 5m | Registration mutations |
| `GET:/api/events/slug/<slug>` | 5m | Registration mutations |
| `GET:/api/events?...` | 2m | `DeletePrefix` on any registration change |
| `GET:/api/guest-registrations?token=<token>` | 2m | Guest registration create/cancel |
| `GET:/api/events/<id>/registrations...` | 2m | Registration status changes |
| `GET:/api/me/registrations:user:<uid>...` | 2m | Registration create/cancel/status |

The cache is **nil-safe**: passing a nil `*cache.Store` bypasses caching entirely without panicking.

### API surface (summary — full table in `docs/TECHNICAL_SPEC.md` §2.1)

- **Public** (`/api`, public rate limiter 10 rps/20 burst): `GET /events[/:id|/slug/:slug]`, `GET /events`, `GET/DELETE /guest-registrations`, `GET /events/:id/volunteer/settings|roles`, social prerender `GET /e/:slug`
- **Guest OTP** (OTP rate limiter 5 req/min): `POST /api/events/:id/register/guest/otp`, `.../otp/verify`
- **Protected** (Clerk JWT → `LoadUser` → 30 rps): `/api/me`, host event CRUD (`POST /events`, `GET /me/events`, `PATCH|DELETE /events/:id`), uploads (`POST /upload/presigned-url` — direct-to-R2), registrations (create/cancel/list/approve/reject/status/check-in), volunteers (apply list, settings/roles/applications/roster admin), `GET /registrations/:id/admin`
- **Idempotency-Key** (UUID header, Redis Lua, 60s in-progress / 5m replay cache, `Idempotent-Replayed: true` on replay) is enabled on: guest otp+register, event create, authenticated register, approve, volunteer apply.

### Guest registration flow (OTP)

1. `POST .../otp` → 6-digit code HMAC-hashed with `OTP_SECRET`, Redis TTL 10m, 60s cooldown per (event, email); email via outbox (best-effort; endpoint 503s if `OTP_SECRET` unset)
2. `POST .../otp/verify` → max 5 tries, sets `guest:verified:{event_id}:{email}` marker (15m)
3. `POST .../register/guest` → consumes marker (one-time), creates registration (`pending` if `requires_approval` else `approved`), issues `access_token` UUID (returned once, 30d TTL), outbox ticket email

### Volunteer programme

`volunteer_settings` (per-event `enabled/description/max_applications`) → `volunteer_roles` (`is_admin` flag) → authenticated `apply` (duplicate-guarded) → organizer approve (inserts roster `volunteers` row, `ON CONFLICT DO NOTHING`) / reject. Approve/reject roles: an admin role is only assignable by the organizer (`ErrVolunteerAdminRoleRestricted`). Partial unique index allows re-apply after cancel/reject.

### Known gaps (tables exist, no code) — see spec §7

- **Payments**: `payments` table + `price_in_paise`/`currency` on events exist; no checkout/processing code at all
- **Kanban**: `kanban_columns`/`kanban_tasks`/`kanban_task_status_log` migrations exist; zero API routes or UI
- **Calendars**: `calendars` table + slug exist, no public calendar pages or endpoints
- **No global admin role, no event scheduler** (statuses only change via explicit cancel)

### Testing with a local database

Auth middleware tests require a real Postgres. Set `TEST_DB_URL` in `backend/.env`:

```
TEST_DB_URL=postgresql://envitoo:envitoo@localhost:5432/envitoo
```

The test harness reads `TEST_DB_URL` first, then falls back to `DB_URL` only if it points at localhost. Remote/Supabase `DB_URL` is never used for tests.

```bash
make dc-up          # start postgres (migrations run on first init)
export PATH="/usr/local/go/bin:$PATH"
cd backend && go test -count=1 ./...
```

Auth-specific tests: `go test -run TestLoadUser ./internal/middleware/`.

### Gotchas

- Put Go 1.26+ on `PATH` before running backend commands: `export PATH="/usr/local/go/bin:$PATH"`.
- Vite proxies `/api` and `/healthz` to `localhost:8080` in dev; leave `VITE_API_BASE_URL` empty in `client/.env`.
- Without real Clerk keys, the UI shell renders but discover/event pages stay empty because Clerk fails to initialize.

### Security implementation notes

*From scan `scan-20260811-e7ee0ca` — see `docs/security/` for full report and remediation.*

**Rate limiter XFF trust** (`server.go:81`): `e.IPExtractor = echo.ExtractIPFromXFFHeader()` trusts all upstream proxies. The backend is only reachable from the frontend container on the Docker internal network; Caddy handles edge filtering. Each real client IP gets its own rate-limit bucket.

**Access token serialization** (`dto/registration.go:113`): `AccessToken` is `*uuid.UUID` with `json:"omitempty"`. The constructor (`NewRegistrationResponse`) never sets it — it stays nil and is omitted from JSON in list/get/admin responses by `omitempty`. Only the two create handlers (`CreateGuestRegistration`, `CreateRegistration`) populate `resp.AccessToken = &token` on the create response. Never expose access tokens in cached or listed payloads.

**Upload limits** (`storage.go`, `upload.go`):
- Extension allowlist: only `.jpg`, `.jpeg`, `.png`, `.webp` are accepted before presigning.
- `Content-Length` in presigned URL: R2 enforces the size cap at the S3 layer. `maxUploadSize = 5 MB` matches the DTO validator.
- `VerifyImage` also checks `Content-Length` via HeadObject for post-upload validation.

**Next param validation** (`client/src/main.tsx`): Blocks `//` (protocol-relative) and `/\` (UNC path) prefixes. `replaceState` is wrapped in try/catch so bad values can never blank the SPA. Protects against latent origin-spoofing risk that would feed cross-origin domains into `window.location.origin`-based ticket URL builders — guest bearer tokens would leak.

**Idempotency key scoping** (`middleware/idempotency.go`): Redis key format is `idem:<METHOD>:<PATH>:<USER_ID>:<CLIENT_KEY>`. User ID is empty for public/guest routes. Cross-route collision is impossible; cross-user replay requires knowing both the victim's key and their user ID.

### Working without Clerk keys (limited mode)

The backend accepts any non-empty `CLERK_SECRET_KEY` at startup; JWT verification only runs on protected routes. Public API endpoints work without real Clerk credentials:

```bash
curl http://localhost:8080/healthz
curl 'http://localhost:8080/api/events?limit=5'
curl http://localhost:8080/api/events/slug/envitoo-launch-meetup
```

The Vite dev proxy forwards these same paths from `:5173` to the backend. Use this to verify DB + API wiring when Clerk keys are unavailable. Auth flows (`/signin`, `/create`, host dashboards) still require valid Clerk keys.
