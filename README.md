# EnviToo

Event management platform: Go API (`backend/`, port 8080) and React/Vite SPA (`client/`, port 5173).

## Running with Docker

The stack runs five services: `backend`, `frontend`, `mail-worker`, `redis`, `nats`
(plus `postgres`). App services are under the `full` profile so `docker compose up`
alone still starts just the infra.

### Local dev

```bash
docker compose --profile full up -d --build
```

Starts postgres, redis, nats, and builds `backend`, `frontend` (Vite dev server),
and `mail-worker` from source.

### Production / EC2 deploy

```bash
export VITE_CLERK_PUBLISHABLE_KEY=...  # or add to a root .env file
IMAGE_TAG=<sha> docker compose --profile full \
  -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

`docker-compose.prod.yml` overrides `backend` and `mail-worker` to pull the
Docker Hub image `hasandotx/envito:<sha>` (built by CI on the `deploy` branch)
instead of building from source; `IMAGE_TAG` defaults to `latest` if unset.
`frontend` is built from `client/` (nginx serving the production bundle, proxying
`/api` and `/healthz` to `backend`). `backend/.env` supplies the secrets
(`CLERK_SECRET_KEY`, `RESEND_API_KEY`, `EMAIL_FROM`, ...), and the
DB/REDIS/NATS URL overrides point at the compose services.

## Database

### TLS is required in production

Production `DB_URL` values must enable TLS:

```
postgresql://<user>:<password>@db.example.com:5432/envitoo?sslmode=require
```

`sslmode=verify-full` is preferred when the CA chain is trusted. When
`APP_ENV=production` (or `DB_REQUIRE_TLS=true`) is set, the backend and worker
fail at startup if the URL contains `sslmode=disable`.

### Pool tuning

Both `cmd/server` and `cmd/worker` open their own pgx pools using the same
env vars (defaults in parentheses):

| Var | Default | Notes |
|-----|---------|-------|
| `DB_MAX_CONNS` | `20` | Per-process cap. api + worker + mailer each open their own pool, so the total is process_count × `DB_MAX_CONNS`. |
| `DB_MIN_CONNS` | `2` | Idle connections kept warm. |
| `DB_MAX_CONN_LIFETIME` | `1h` | Recycle connections to dodge long-lived connection stalls. |
| `DB_MAX_CONN_IDLE_TIME` | `30m` | Close idle connections after this long. |
| `DB_HEALTH_CHECK_PERIOD` | `1m` | Background health checks. |
| `DB_DISABLE_PREPARED_STATEMENTS` | `false` | `true` behind PgBouncer (transaction mode). |
| `DB_REQUIRE_TLS` | auto in prod | Rejects `sslmode=disable` URLs. |

**Sizing `DB_MAX_CONNS`:** check the database's `max_connections`
(`SHOW max_connections`), subtract the headroom other workloads need
(monitoring, pgbouncer, manual queries), then divide the rest across
processes. At startup the backend logs a warning when a single pool's
`MaxConns >= 80%` of `max_connections`.

### Connection hygiene

Service methods run all DB writes (domain row + `email_outbox` row) in one
transaction; `internal/helper.RollbackTx` is the standard deferred rollback
that suppresses the harmless "transaction already closed" error and logs real
rollback failures.

## See also

- `Makefile` — standard dev commands (`make dev`, `make dc-up`, `make check`, etc.)
- `AGENTS.md` — environment setup and repo conventions
