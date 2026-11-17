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

Push to the `deploy` branch to build and deploy the Docker images to EC2. The
production Compose stack runs PostgreSQL, Redis, NATS, the API, frontend, email
workers, and Caddy on one instance. PostgreSQL, Redis, and NATS data use Docker
volumes; PostgreSQL is not published on a host port. The EC2 security group
should allow inbound TCP 80/443 and restrict SSH (22) to trusted sources.

Configure these GitHub Actions secrets:

- Database: `POSTGRES_PASSWORD` (URL-safe alphanumeric; generated automatically
  by the deployment setup)
- App: `CLERK_SECRET_KEY`, `OTP_SECRET` (generated automatically),
  `RESEND_API_KEY`, `EMAIL_FROM`,
  `APP_PUBLIC_URL`, `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`,
  `R2_SECRET_ACCESS_KEY`, `R2_BUCKET_NAME`, `R2_PUBLIC_URL`,
  `VITE_CLERK_PUBLISHABLE_KEY`
- Image registry: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`

The deploy job uses GitHub OIDC to assume a narrowly scoped AWS role, uploads an
encrypted bundle to a private S3 bucket, and runs the deployment over Systems
Manager. Configure repository Actions variables `AWS_ROLE_TO_ASSUME`,
`AWS_REGION`, `EC2_INSTANCE_ID`, `DEPLOY_ARTIFACT_BUCKET`, and
`R2_CREDENTIALS_FRESH=yes`. The instance has Docker Engine and Compose v2
installed; SSH remains limited to the workstation key and source IP.

Set `APP_PUBLIC_URL` to `https://app.mnah.dev` and point that hostname at the
instance before expecting Caddy HTTPS to become active. Back up the
PostgreSQL EBS-backed Docker volume; container volumes alone do not protect
against loss of the instance's disk. Deployment bundles expire from S3 after
seven days.

## Database

### Database TLS

Externally hosted production databases must enable TLS:

```
postgresql://db.example.com:5432/envitoo?sslmode=require
```

`sslmode=verify-full` is preferred when the CA chain is trusted. For this
single-instance deployment, PostgreSQL is only reachable on the private Docker
network and the connection uses `sslmode=disable`; do not publish its port.
When `APP_ENV=production` (or `DB_REQUIRE_TLS=true`) is set, the backend and
worker reject that local connection mode.

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
