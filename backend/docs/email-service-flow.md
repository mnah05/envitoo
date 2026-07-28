# Email delivery pipeline (outbox → NATS → Resend)

## Overview

```
API / services
  └─ INSERT email_outbox (PENDING) in the same TX as the domain write
        ↓
cmd/worker (outbox drainer)
  └─ FetchPending SKIP LOCKED → PublishAsync to JetStream (MsgID=outbox_id)
  └─ Mark PUBLISHED / bump retries → FAILED
        ↓
NATS JetStream stream EMAILS (subjects emails.>)
        ↓
cmd/mailer (durable consumer email-resend)
  └─ Render HTML+text templates
  └─ Resend send (idempotency key = outbox_id)
  └─ Ack / Nak / Term
```

Producers never call Resend directly. The mailer never needs Postgres for send content — the JetStream payload is self-contained.

## Processes

| Binary | Role | Required env |
|---|---|---|
| `cmd/server` | Inserts outbox rows | `DB_URL`, `CLERK_*`, Redis/`OTP_SECRET` for guest OTP |
| `cmd/worker` | Drains outbox → NATS | `DB_URL`, `NATS_URL`, `WORKER_*` |
| `cmd/mailer` | NATS → Resend | `NATS_URL`, `RESEND_API_KEY`, `EMAIL_FROM`, `APP_PUBLIC_URL` |

Local run (after NATS + Redis + Postgres are up):

```bash
cd backend
go run ./cmd/server
go run ./cmd/worker
go run ./cmd/mailer
```

## JetStream

On startup, both worker and mailer ensure:

- Stream `EMAILS`, subjects `emails.>`, work-queue retention
- Mailer durable consumer `email-resend`, explicit ack, max deliver 10

## Templates (9)

| Template | Topic |
|---|---|
| `otp` | `emails.otp` |
| `ticket_confirm` | `emails.ticket.confirmed` |
| `ticket_cancel` | `emails.ticket.cancelled` |
| `ticket_status_change` | `emails.ticket.status_changed` |
| `application_confirm` | `emails.application.confirmed` |
| `application_cancel` | `emails.application.cancelled` |
| `volunteer_status_change` | `emails.volunteer.status_changed` |
| `check_in` | `emails.checkin` |
| `event_cancellation` | `emails.event.cancelled` |

Rendering lives in `internal/email` (`embed` HTML + text). `APP_PUBLIC_URL` builds ticket (`/ticket?token=`) and home (`/home`) links.

## Idempotency

- Drainer publish uses JetStream `MsgID = outbox_id`
- Mailer Resend calls use `IdempotencyKey = outbox_id`

Redeliveries should not create duplicate inbox mail under normal Resend semantics.

## Failure modes

| Stage | Behavior |
|---|---|
| Drainer publish fail | Row stays PENDING; retries until `WORKER_MAX_RETRIES` then FAILED |
| Mailer render unknown template / bad JSON | Ack (poison) — do not infinite loop |
| Mailer Resend 4xx (permanent) | Term message |
| Mailer Resend 5xx / network | Nak for redelivery |

Note: once the drainer marks a row PUBLISHED, delivery retries are NATS-only. Monitor mailer logs for Term'd OTP/ticket sends.

## Guest OTP + email

See also `guest-email-otp.md`. Request OTP inserts `template=otp` into the outbox; delivery depends on worker + mailer + Resend being healthy.
