# AGENTS.md

## Cursor Cloud specific instructions

EnviToo is an event management platform with a **Go API** (`backend/`, port 8080) and **React/Vite SPA** (`client/`, port 5173). See the root `Makefile` for standard commands.

### System dependencies (one-time on a fresh VM)

These are **not** in the startup update script:

- **Go 1.26+** at `/usr/local/go` (the repo's `go.mod` requires 1.26.4; system `/usr/bin/go` may be older)
- **PostgreSQL** and **Redis** via `apt` (`postgresql`, `redis-server`)
- Create a local DB user/database, e.g. `envitoo` / `envitoo` on `localhost:5432`
- Apply SQL migrations manually: run every file in `backend/supabase/migrations/` in filename order with `psql` (there is no `docker-compose`; Supabase CLI is optional and expects Docker)

### Environment files (required to run the app)

Copy templates and fill in secrets (never commit `.env` files):

| File | Required vars |
|------|----------------|
| `backend/.env` | `DB_URL`, `CLERK_SECRET_KEY`; optional `REDIS_URL`, `OTP_SECRET` |
| `client/.env` | `VITE_CLERK_PUBLISHABLE_KEY` |

Clerk keys come from the Clerk dashboard or `npx clerk@1.5.0 env pull` after `clerk auth login` + `clerk link`. **Valid Clerk keys are required** for the frontend to load public event data (the API client uses `useAuth()` from Clerk even for unauthenticated routes).

### Starting services

```bash
export PATH="/usr/local/go/bin:$PATH"
sudo service postgresql start
sudo service redis-server start
make dev   # backend :8080 + frontend :5173
```

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
