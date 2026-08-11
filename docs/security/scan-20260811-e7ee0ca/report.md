# Security Scan Report — EnviToo

- **Scan ID:** `scan-20260811-e7ee0ca`
- **Date (UTC):** 2026-08-11T12:54:11Z
- **Target:** `git@/Users/mnah/code-space/envito` @ `e7ee0ca`
- **Skill:** `security-scan` (headless adaptation; original host-only plumbing removed)

## 1. Method and disclosure

This scan ran in **degraded mode**. The skill's baseline-auditor + independent-investigator
model requires delegation, but agent launches were repeatedly cancelled or returned empty
results. Per the skill's documented fallback, the **parent performed the baseline audit and
every focused investigation sequentially**, and disclosed here that an *independent* baseline
was unavailable. Findings were validated directly against source. Token/usage figures in the
manifest reflect parent-side operations only.

**Scope:** `backend/` (Go 1.26, Echo v4, pgx/v5 + sqlc, Redis, NATS JetStream email outbox,
Cloudflare R2), `client/` (React/Vite SPA), and infra (`docker-compose*.yml`, `Caddyfile`,
`Dockerfile`, `Makefile`, `.github/workflows`). sqlc-generated files and `node_modules` excluded.

**No SECURITY.md, threat model, or security policy was supplied or present in the repo.**

## 2. Summary

| Severity | Count | IDs |
|----------|-------|-----|
| Critical | 0 | — |
| High     | 0 | — |
| Medium   | 3 | ENV-2026-01, ENV-2026-02, ENV-2026-03 |
| Low      | 2 | ENV-2026-04, ENV-2026-05 |
| Info     | 4 | ENV-2026-06, ENV-2026-07, ENV-2026-08, hardening notes |

## 3. Findings

### ENV-2026-01 (Medium) — Rate limiters collapse to global buckets behind the proxy chain
`backend/internal/server/server.go:81` uses `echo.ExtractIPFromXFFHeader(echo.TrustLoopback(true))`.
Production traffic is Caddy → frontend container → backend, so the backend's peer is a non-loopback
container IP and the XFF header is ignored; every request shares one `c.RealIP()`. The public
(10 rps / 20 burst) and OTP (5/min) limiters in `backend/internal/middleware/ratelimit.go` become
single global buckets → one attacker can 429 the whole platform and spam OTP emails (~5/min) to
arbitrary addresses. **Fix:** trust the configured proxy set, add edge limiting, move OTP counters to Redis.

### ENV-2026-02 (Medium) — Access tokens always serialized in responses
`dto/registration.go:113` declares `AccessToken uuid.UUID` with `json:",omitempty"`, which never
omits a `[16]byte` array; `NewRegistrationResponse` sets it unconditionally, contradicting the DTO's
own Note 10. Staff list endpoints therefore return every attendee's bearer token; a volunteer admin
(or anyone who leaks a staff-list payload) can cancel arbitrary guest registrations through the public
`DELETE /api/guest-registrations?token=` endpoint. **Fix:** `*uuid.UUID`, populate only on create.

### ENV-2026-03 (Medium) — Unbounded direct-to-R2 uploads, user-controlled suffix, orphaned objects
`storage.go:91` issues a 5-min presigned PUT with no size cap; key suffix comes from the client's
filename; covers are nullified but never deleted. Content-Type is signed (image/jpeg|png|webp) so
MIME confusion is mitigated, but any authenticated user can fill the bucket / run up cost, and
objects accumulate. **Fix:** size cap + suffix allowlist + lifecycle/delete-on-nullify.

### ENV-2026-04 (Low) — `next` param crashes the SPA (blank page)
`client/src/main.tsx:12` accepts `//evil.com` (passes `startsWith("/")`); `history.replaceState`
throws `SecurityError` on the cross-origin URL at module scope → blank page for any clicking user.
Keep the fix future-proof: reject `//` and `/\` so origin-spoofing can never reach the
`window.location.origin`-based ticket/event URL builders. **Fix:** same-origin path check + try/catch.

### ENV-2026-05 (Low) — Idempotency keys not bound to user/route; response replay can leak PII/tokens
`idempotency.go:111` keys Redis on `idem:<client-key>` with a body-only hash; bodyless routes share
a hash; `done` entries replay stored bodies (incl. guest `access_token` from registration-create)
for 5 min to any caller with the same key. Requires the victim's UUID key — low likelihood.
**Fix:** scope key by caller + route; don't persist sensitive fields.

### Info findings
- **ENV-2026-06:** No `SECURITY.md`; Caddyfile sends no HSTS/CSP/X-Frame-Options; SPA has no CSP
  and no `Referrer-Policy` (the `/ticket?token=` page should not leak the token in Referer).
- **ENV-2026-07:** `/healthz` exposes Go version, git revision, uptime (optional token); CORS is `*`
  (acceptable for token SPA; no cookie auth).
- **ENV-2026-08:** DTO `url` validator accepts any scheme (`javascript:`); neutralized by React 16.9+
  sanitization and (for covers) `VerifyImage` R2-host checks.
- Rate limiter is in-process (resets on restart) — documented trade-off, note only.

## 4. Verified-sound areas (no findings)

- **AuthN:** Clerk JWT + pre-warmed JWKS; provisioning race-safe via `ON CONFLICT DO NOTHING`;
  guest linking scoped by verified email; registration email always from session, never client-supplied.
- **Guest OTP:** `crypto/rand` 6-digit; HMAC-SHA256 only (no plaintext in Redis); constant-time
  compare; 5-try limit; 10 min TTL; verified marker single-use; per-(event,email) 60s cooldown.
- **AuthZ:** staff checks run before cached-staff payloads are served; user-scoped queries; organizer-only
  `UpdateStatus`; volunteer admin-role restricted to organizer; capacity guarded with `SELECT FOR UPDATE`.
- **Email pipeline:** `html/template` auto-escaping; Resend JSON API (no SMTP injection); transactional
  outbox; JetStream `MsgID` idempotency; Ack/Nak/Term poison handling.
- **Client:** no `dangerouslySetInnerHTML`/`eval`/`DOMParser` sinks; guest token lives only in the URL.
- **Secrets:** `.env*` gitignored (`.env.local` untracked); CI runs gitleaks; GitHub Actions use
  `secrets.*` references only.
- **SQL:** sqlc/parameterized only; no string-concatenated queries found.

## 5. Recommended priority

1. Fix ENV-2026-01 (proxy trust) — one-line change with broadest availability impact.
2. Fix ENV-2026-02 (token serialization) — realizes a documented control that is currently ineffective.
3. Fix ENV-2026-03 (upload caps) and ENV-2026-04 (next-param) — cheap, remove abuse + crash paths.
4. Add SECURITY.md + security headers (ENV-2026-06).

*Full machine-readable artifacts: `scan-manifest.json`, `findings.json`, `coverage.json` (same directory).*
