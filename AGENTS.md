# AGENTS.md

## Cursor Cloud specific instructions

EnviToo is an event management platform with a **Go API** (`backend/`, port 8080) and **React/Vite SPA** (`client/`, port 5173). See the root `Makefile` for standard commands.

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

Optional email pipeline: `make worker` and `make mailer` (need NATS + Resend).

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

### Working without Clerk keys (limited mode)

The backend accepts any non-empty `CLERK_SECRET_KEY` at startup; JWT verification only runs on protected routes. Public API endpoints work without real Clerk credentials:

```bash
curl http://localhost:8080/healthz
curl 'http://localhost:8080/api/events?limit=5'
curl http://localhost:8080/api/events/slug/envitoo-launch-meetup
```

The Vite dev proxy forwards these same paths from `:5173` to the backend. Use this to verify DB + API wiring when Clerk keys are unavailable. Auth flows (`/signin`, `/create`, host dashboards) still require valid Clerk keys.
