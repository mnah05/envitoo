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

## See also

- `Makefile` — standard dev commands (`make dev`, `make dc-up`, `make check`, etc.)
- `AGENTS.md` — environment setup and repo conventions
