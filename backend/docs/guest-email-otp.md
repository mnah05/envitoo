# Plan: Guest Email OTP (Request + Verify Only)

## Context

Guests must prove email ownership before registration. This plan covers **only**:

1. Request OTP endpoint  
2. Verify OTP endpoint  

**Out of scope**

- Guest registration creation / changes to `CreateGuest`
- Registration verification gates on create
- Ticket generation / ticket creation
- Worker/drainer changes
- NATS changes
- New email-sending path (API never sends mail directly)
- Duplicate registration checks, capacity checks, approval rules (stay in `CreateGuest`)

### Separation of concerns

```
Request OTP:     validate event → generate OTP → SET Redis OTP → insert email_outbox (tx) → on DB fail DELETE Redis OTP
Verify OTP:      prove email ownership → write short-lived verified marker
CreateGuest:     registration rules + create registration  (later; not this task)
```

Email delivery already exists:

```
API inserts email_outbox (PENDING)
        ↓
cmd/worker/drainer publishes to NATS JetStream
        ↓
external consumer sends mail
```

---

## Existing outbox contract (inspected — do not reinvent)

### Table (`email_outbox`)

| Column | Notes |
|--------|--------|
| `id` | UUID PK, default `gen_random_uuid()` |
| `recipient` | TEXT NOT NULL |
| `template` | TEXT NOT NULL |
| `template_data` | JSONB NOT NULL default `{}` |
| `topic` | TEXT NOT NULL default `emails.unknown` |
| `status` | `PENDING` \| `PUBLISHED` \| `FAILED` |
| `retries`, `last_error`, `created_at`, `published_at`, `failed_at` | worker-managed |

### Worker template / topic (already defined)

`cmd/worker/drainer.go`:

- Template: `"otp"` (`TemplateOTP`)
- Topic: `"emails.otp"`

Published message shape (worker builds this; API only inserts the row):

```json
{
  "outbox_id": "<uuid>",
  "recipient": "<email>",
  "template": "otp",
  "template_data": { ... }
}
```

### Producer status today

- **No** API `INSERT INTO email_outbox` yet  
- **No** sqlc model/query for outbox (worker uses raw SQL)  
- This task is the first API producer for OTP rows  

### Outbox insert to add

```sql
-- name: CreateEmailOutbox :one
INSERT INTO email_outbox (recipient, template, template_data, topic)
VALUES ($1, $2, $3, $4)
RETURNING *;
```

- `template` = `"otp"`  
- `topic` = `"emails.otp"`  
- `template_data` = JSON for email render (plaintext OTP **only** here, never in Redis)

**Assumption (no consumer schema in this repo):**  
`template_data` ≈ `{"otp":"123456","name":"...","event_title":"...","event_id":"..."}`  
Do not invent alternate template names or topics.

---

## Architecture fit

| Layer | Where | OTP work |
|-------|--------|----------|
| Handler | `internal/server/registration.go` | HTTP bind/validate/map errors |
| Routes | `internal/server/server.go` public `/api` group | Wire two POST routes |
| Service | `internal/service/registration.go` | OTP generate/store/verify + outbox insert |
| Config | `internal/config/config.go` | `OTP_SECRET` for HMAC |
| DB | sqlc (`sql/` → `internal/db`) | Outbox insert **only** (no registration-exists queries) |
| Redis | `internal/cache` | Hard-fail ops for OTP state (not soft cache) |
| Errors | `internal/service/errors.go` | OTP / dependency sentinels |
| DTOs | `internal/dto/registration.go` | Request/response types |

- No new packages. No repository layer (project uses sqlc only).  
- Keep OTP methods cohesive and self-contained on `RegistrationService` so they can later move into a `GuestVerificationService` if more verification flows appear. For now, `RegistrationService` is acceptable.

---

## Endpoints

Public (same group as guest registration + existing IP rate limiter):

| Method | Path | Body |
|--------|------|------|
| `POST` | `/api/events/:id/register/guest/otp` | `name`, `email` |
| `POST` | `/api/events/:id/register/guest/otp/verify` | `email`, `otp` |

### DTOs

```go
type RequestGuestOTPRequest struct {
    Name  string `json:"name" validate:"required,min=1,max=200"`
    Email string `json:"email" validate:"required,email"`
}

type RequestGuestOTPResponse struct {
    Status           string `json:"status"` // "otp_sent"
    ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type VerifyGuestOTPRequest struct {
    Email string `json:"email" validate:"required,email"`
    OTP   string `json:"otp" validate:"required,len=6,numeric"`
}

type VerifyGuestOTPResponse struct {
    Status           string `json:"status"` // "verified"
    ExpiresInSeconds int    `json:"expires_in_seconds"`
}
```

---

## OTP hashing (HMAC, not plain SHA-256)

A 6-digit OTP has only ~1e6 possibilities. Plain `SHA256(eventID|email|otp)` can be offline-brute-forced if Redis is leaked.

**Store only:**

```
HMAC_SHA256(OTP_SECRET, eventID|normalizedEmail|otp)
```

- Secret from config/env: `OTP_SECRET` (required for OTP endpoints; inject into service)  
- Message: `eventID.String() + "|" + normalizedEmail + "|" + otp`  
- Store hex/base64 of the HMAC digest in Redis field `hash`  
- Verify with the same HMAC + `subtle.ConstantTimeCompare`  
- **Never** store plaintext OTP in Redis  

---

## Redis design

HTTP cache uses soft-fail `Store` (`GET:/api/...` keys). **OTP must not use silent no-ops.**

Extend `internal/cache` with hard-fail helpers:

- `Available()`
- `GetBytes` / `SetBytes` (error if Redis down)
- `SetNX` (resend cooldown)
- `Incr` (attempt counter)
- `Expire` (required after first `INCR` — see below)
- strict delete that returns errors

### Keys

| Key | TTL | Value |
|-----|-----|--------|
| `guest:otp:{event_id}:{email}` | ~10m | `{"hash","name","email"}` — **HMAC only** (no attempts field) |
| `guest:otp:attempts:{event_id}:{email}` | ~10m (**must set explicitly**) | integer counter via **`INCR`** |
| `guest:otp:cooldown:{event_id}:{email}` | ~60s | resend lock (`SET NX`) |
| `guest:verified:{event_id}:{email}` | ~15m | `{"email","name","verified_at"}` |

- Email in keys: `strings.ToLower(strings.TrimSpace(...))`
- Max attempts: **5** (on `INCR` result > max → delete OTP + attempts keys)  
- On success: delete OTP + attempts keys (single-use), write verified marker  
- On new OTP request: overwrite OTP key; **reset attempts** (delete attempts key so the next verify starts a fresh counter with a fresh TTL)

### Why separate `INCR` for attempts

Storing `attempts` inside the OTP JSON requires GET → mutate → SET, which races under concurrent verify requests. A dedicated attempts key with Redis `INCR` is atomic and barely more code.

### Critical: `INCR` does **not** set TTL

`INCR` on a brand-new key creates a **permanent** key with no expiry. If you never call `Expire`, after 5 failed attempts that `event+email` stays locked out indefinitely instead of recovering after ~10m.

**Required pattern on every verify attempt:**

```go
count, err := redis.Incr(ctx, attemptsKey)
// ...
if count == 1 {
    // First increment created the key — attach TTL explicitly.
    if err := redis.Expire(ctx, attemptsKey, 10*time.Minute); err != nil {
        // handle / best-effort log; do not leave key without TTL
    }
}
if count > maxAttempts { // e.g. 5
    // delete OTP + attempts; return invalid
}
```

Do not assume “we have TTLs everywhere” covers this key unless `Expire` runs when `count == 1`.

### OTP replacement (overwrite)

When a new OTP is requested for the same `event_id + email`, Redis `SET` on `guest:otp:{event_id}:{email}` **overwrites** the previous value (and refreshes TTL). Also reset the attempts key for that event+email.

Example:

- 10:00 → OTP `123456` stored  
- 10:01 → OTP `654321` overwrites key  

Only `654321` remains valid. Previous OTP is automatically invalid. No extra delete step required beyond normal `SET` (+ attempts reset).

---

## Service flow

### `RequestGuestOTP`

OTP proves email ownership only. **Do not** run capacity, duplicate-registration, or approval logic here.

1. Normalize email  
2. Require Redis → else `ErrDependencyUnavailable` (503)  
3. Require `OTP_SECRET` configured → else 503 / config error  
4. **Validate event only** (`GetEventByID`):  
   - missing → `ErrEventNotFound` (404)  
   - not `published` → `ErrEventNotOpen` (409)  
   - **No** capacity check  
   - **No** `ExistsActiveRegistrationByEventEmail` / duplicate check  
5. Resend cooldown via `SET NX` on cooldown key → fail → `ErrOTPRateLimited` (429)  
6. Generate 6-digit OTP (`crypto/rand` over `0..999999`), format with **`fmt.Sprintf("%06d", n)`** so leading zeros are preserved (e.g. `42` → `"000042"`). HMAC and email payload must use this same zero-padded string.  
7. **SET Redis OTP state first** (`hash`, `name`, `email` only), overwriting any prior OTP for this event+email; **reset attempts key** (delete or zero + TTL ~10m)  
8. **Then** `BEGIN` DB transaction → `INSERT email_outbox` (`template=otp`, `topic=emails.otp`, plaintext zero-padded OTP in `template_data` only) → `COMMIT`  
9. **If DB insert/commit fails:** `DELETE` Redis OTP key + attempts key (and release cooldown if held) so no orphaned verifiable OTP remains without an email path; return error  
10. If Redis SET fails before DB: return 503; do not insert outbox  
11. Return `{status:"otp_sent", expires_in_seconds}` — **never return OTP in HTTP body**

#### Ordering (important)

```
Generate OTP
    │
    v
SET Redis OTP          (overwrite previous OTP for same event+email)
    │
    v
BEGIN DB transaction
    INSERT email_outbox
COMMIT
    │
    ├─ success → done
    │
    └─ DB fails → DELETE Redis OTP + attempts key  (cleanup so verify cannot succeed without email queued)
```

**Why Redis then outbox + cleanup:**

- Store OTP first so verify state exists only after a successful write path is attempted.  
- If outbox insert fails after Redis SET, **delete the Redis OTP** (and attempts key) so the user does not hold a valid OTP that can never arrive by email.  
- User can safely request again (after cooldown if still active).  
- Prefer a transaction around the outbox insert even if it is a single statement, so commit failure is explicit and cleanup is reliable.

**Failure matrix**

| Step that fails | Action | User impact |
|-----------------|--------|-------------|
| Redis SET | Abort; no outbox insert | Retry request |
| DB insert/commit | DELETE Redis OTP + attempts (+ optional cooldown clear) | Retry request; no usable OTP left in Redis |
| Both succeed | Email queued; OTP verifiable | Happy path |

### `VerifyGuestOTP`

1. Normalize email; require Redis + secret  
2. Load OTP key → missing/expired → generic `ErrOTPInvalid`  
3. `INCR` attempts key; **if `count == 1`, `Expire(attemptsKey, 10m)`** (INCR alone does not set TTL — see Redis section). If `count > max` (5) → delete OTP + attempts → `ErrOTPInvalid`  
4. HMAC submitted OTP (as provided 6-char string); constant-time compare to stored digest  
5. Mismatch → return `ErrOTPInvalid` (counter already advanced by `INCR`; no JSON rewrite race)  
6. Match → **delete OTP + attempts keys** (single-use); set `guest:verified:{event_id}:{email}` with `email`, `name`, `verified_at`  
7. Return `{status:"verified", expires_in_seconds}`  

**Do not create registration rows. Do not apply registration business rules.**

---

## Handler error mapping

| Sentinel | HTTP |
|----------|------|
| `ErrEventNotFound` | 404 |
| `ErrEventNotOpen` | 409 |
| `ErrOTPRateLimited` | 429 |
| `ErrOTPInvalid` | 400 `"invalid or expired otp"` (uniform) |
| `ErrDependencyUnavailable` | 503 (Redis / missing OTP secret) |
| other | 500 + log |

Note: `ErrEventFull` / `ErrConflict` are **not** produced by OTP endpoints (registration-only).

Use existing handler style: bind → validate → service → `errors.Is` → JSON.

---

## Security

| Concern | Approach |
|---------|----------|
| Offline OTP brute-force if Redis leaked | HMAC-SHA256 with `OTP_SECRET` |
| Online brute force | Max 5 attempts via atomic `INCR` key, then delete OTP + attempts |
| Attempts key never expires | Always `Expire` when `INCR` returns `1`; reset (delete) attempts key on new OTP request |
| Expiry | Redis TTL on OTP key (~10m) |
| Reuse | Delete OTP key on successful verify |
| Replacement | New request `SET` overwrites prior OTP for same event+email |
| Resend spam | Cooldown key + existing public IP rate limit |
| Plaintext | HMAC only in Redis; OTP only in outbox for email render |
| Enumeration | Uniform verify error message |

---

## Files to change

| File | Change |
|------|--------|
| `docs/guest-email-otp.md` | This plan |
| `internal/config/config.go` | Load `OTP_SECRET` |
| `internal/cache/redis.go`, `store.go` | Hard-fail Get/Set/SetNX/Incr |
| `sql/email_outbox.sql` | `CreateEmailOutbox` only |
| `internal/db/*` | `sqlc generate` |
| `internal/dto/registration.go` | OTP DTOs |
| `internal/service/errors.go` | OTP / dependency sentinels |
| `internal/service/registration.go` | Cache + secret deps; isolated OTP methods |
| `internal/server/server.go` | Inject deps; routes |
| `internal/server/registration.go` | Two handlers |
| `internal/service/*_otp*_test.go` | Unit / miniredis tests (HMAC, overwrite, attempts) |
| `internal/docs/api-reference.md` | Document endpoints |

**Do not add:** `ExistsActiveRegistrationByEventEmail` or any registration-rule SQL for OTP.

**Do not touch:** `cmd/worker/*`, `CreateGuest`, NATS config.

---

## Implementation order

1. Config: `OTP_SECRET`  
2. Cache hard-fail helpers  
3. sqlc outbox query + generate  
4. DTOs + error sentinels  
5. Service `RequestGuestOTP` / `VerifyGuestOTP` (HMAC; Redis-then-outbox with DELETE on DB fail; overwrite)  

6. Handlers + routes + DI  
7. Tests  
8. API reference note  

---

## Verification

- `go test ./internal/cache/... ./internal/service/...`  
- `go build ./...`  
- Request OTP → `email_outbox` row `template=otp`, `topic=emails.otp`, `status=PENDING`  
- Redis holds HMAC only (no plaintext OTP)  
- Second request for same event+email invalidates previous OTP  
- Verify with latest OTP → `guest:verified:...` set; OTP key deleted  
- Old OTP after resend → 400  
- Wrong OTP / expiry / max attempts → 400 uniform message  
- After first failed verify, attempts key has a TTL (not permanent); after OTP TTL, lockout does not stick forever  
- Resend within cooldown → 429  
- Redis down or missing secret → 503  
- No capacity/duplicate errors from OTP endpoints  
- No new rows in `registrations`  
- Worker/drainer unchanged  

---

## After implementation (report)

1. Files changed  
2. OTP flow explanation  
3. Redis key structure  
4. Outbox payload format used  
5. Assumptions  

---

## Assumptions

1. OTP methods live on `RegistrationService` for now, structured so a later extract to `GuestVerificationService` is straightforward.  
2. Redis required for OTP even if optional elsewhere.  
3. `OTP_SECRET` is a required server secret for HMAC; not committed; loaded from env.  
4. Template/topic strings match worker: `"otp"` / `"emails.otp"`.  
5. `template_data` fields as above until consumer schema is documented in-repo.  
6. Defaults: OTP 10m, verified 15m, cooldown 60s, max 5 attempts (`INCR` key).  
7. Attempts key TTL is **not** free: always `Expire` when `INCR` returns `1`, or the key is permanent.  
8. OTP string is always 6 characters via `%06d` (leading zeros preserved in HMAC, email, and verify input).  
9. Request OTP validates event existence + published status only — not capacity or duplicates.  
10. Order: Redis SET OTP → DB outbox insert → on DB failure DELETE Redis OTP + attempts.  
11. `CreateGuest` remains unguarded until a follow-up reads `guest:verified:...`.  
