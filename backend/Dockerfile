FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /server ./cmd/server \
    && CGO_ENABLED=0 go build -o /mailer ./cmd/mailer \
    && CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /server /usr/local/bin/server
COPY --from=builder /mailer /usr/local/bin/mailer
COPY --from=builder /worker /usr/local/bin/worker
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Runtime configuration is injected via environment variables (DB_*, CLERK_*,
# R2_*, REDIS_URL, NATS_URL, ...) — `docker run -e`, compose `environment:` /
# `env_file:`, or an env file mounted at /app/.env (see docker-entrypoint.sh).
WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["server"]
