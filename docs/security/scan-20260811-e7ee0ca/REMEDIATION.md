# Remediation Report — Security Scan `scan-20260811-e7ee0ca`

- **Date fixed:** 2026-08-11
- **Commit:** pending (uncommitted)
- **Scan ref:** `docs/security/scan-20260811-e7ee0ca/report.md`

---

## Fixes Applied

### ENV-2026-01 (Medium) — Rate limiters collapse to global buckets behind proxy chain

**File:** `backend/internal/server/server.go:81`

**Change:**
```diff
- e.IPExtractor = echo.ExtractIPFromXFFHeader(echo.TrustLoopback(true))
+ e.IPExtractor = echo.ExtractIPFromXFFHeader() // trusts all upstream proxies (production: Caddy → frontend → here)
```

**Rationale:** `TrustLoopback(true)` only trusts XFF from loopback peers. In production, Caddy → frontend container → backend, so the backend's immediate peer is a Docker bridge IP (non-loopback). The XFF header was ignored and `c.RealIP()` returned the single frontend-container IP for every request, collapsing the public (10 rps) and OTP (5/min) rate limiters into single global buckets. One attacker could 429 the entire platform.

`TrustAllProxies()` (no arguments = default) is appropriate because the backend is only reachable from the frontend container on the internal Docker network — no direct external access. Caddy handles the edge filtering.

**Verification:** `go test ./internal/server ./internal/middleware/` — all pass.

---

### ENV-2026-02 (Medium) — Access tokens always serialized in responses

**Files:**
- `backend/internal/dto/registration.go:113` — changed `AccessToken uuid.UUID` to `AccessToken *uuid.UUID`
- `backend/internal/dto/registration.go:145` — removed unconditional `AccessToken: r.AccessToken` from `NewRegistrationResponse`
- `backend/internal/server/registration.go:136-138` — `CreateGuestRegistration` now sets `resp.AccessToken = &token` after construction
- `backend/internal/server/registration.go:249-251` — `CreateRegistration` now sets `resp.AccessToken = &token` after construction

**Rationale:** `uuid.UUID` is a `[16]byte` value type; `json:"omitempty"` never triggers because the zero value serializes as `"00000000-..."` (a non-empty JSON string). `NewRegistrationResponse` unconditionally copied the token from the DB row into every response — creates, list (GET `/api/events/:id/registrations`), gets, cancels, approve/reject, check-in. Staff list endpoints returned every attendee's bearer token; anyone who leaked a staff-list payload could cancel arbitrary guest registrations via `DELETE /api/guest-registrations?token=`.

After the fix:
- The constructor no longer sets `AccessToken` — it stays `nil` by default, which `omitempty` correctly omits from JSON.
- Only the two create handlers (`CreateGuestRegistration`, `CreateRegistration`) populate the token on the response.

**Verification:** `go test ./internal/server/` — all pass.

---

### ENV-2026-03 (Medium) — Unbounded direct-to-R2 uploads + user-controlled suffix

**Files:**
- `backend/internal/server/upload.go:30-33` — extension allowlist validation added
- `backend/internal/server/upload.go:36` — passes `req.Size` to `GeneratePresignedURL`
- `backend/internal/service/storage.go:20` — `const maxUploadSize = 5 * 1024 * 1024`
- `backend/internal/service/storage.go:91` — `GeneratePresignedURL` accepts `size int64` param, rejects >5MB
- `backend/internal/service/storage.go:106-111` — `ContentLength: aws.Int64(size)` added to presigned PutObjectInput
- `backend/internal/service/storage.go:148-150` — post-upload Content-Length check in `VerifyImage`

**Rationale:** The DTO validated `Size` ≤5MB client-side, but the presigned URL had no `Content-Length` constraint — a client could claim 5MB, upload 5GB. User-provided filename extensions passed through unchecked (though `VerifyImage` rejected non-image Content-Types). No cleanup existed for orphaned objects when covers were nullified.

After the fix:
- Server-side extension allowlist rejects non-image extensions (`.jpg`, `.jpeg`, `.png`, `.webp`) before presigning.
- `Content-Length` is signed into the presigned URL → R2 rejects oversized uploads at the S3 layer.
- `VerifyImage` performs a post-upload size check via `HeadObject.ContentLength`.
- `maxUploadSize` const (5MB) matches the DTO validator's `max=5242880`.

**Verification:** `go test ./internal/service/ ./internal/server/` — all pass.

---

### ENV-2026-04 (Low) — Next param crashes the SPA (blank page)

**File:** `client/src/main.tsx:11-22`

**Change:**
```diff
- const next = new URLSearchParams(window.location.search).get("next");
- if (next && next.startsWith("/")) {
-   window.history.replaceState(null, "", next);
- }
+ const next = new URLSearchParams(window.location.search).get("next");
+ if (
+   next &&
+   next.startsWith("/") &&
+   !next.startsWith("//") &&
+   !next.startsWith("/\\")
+ ) {
+   try {
+     window.history.replaceState(null, "", next);
+   } catch {
+     // Malformed/cross-origin next value — silently ignored.
+   }
+ }
```

**Rationale:** `next.startsWith("/")` alone passed protocol-relative URLs like `//evil.com`. Browsers resolve these as cross-origin, and `history.replaceState` throws `SecurityError` at module scope (no try/catch) → blank page for any clicking user. Windows-style UNC paths (`/\evil.com`) have the same effect in some browsers.

After the fix:
- Block `//` (protocol-relative URLs) and `/\` (UNC paths).
- Wrap `replaceState` in try/catch as defense-in-depth — a bad value can never blank the app.
- This also prevents a latent risk: if the validation were ever relaxed, a cross-origin `next` value would feed into `window.location.origin`-based ticket URL builders (`TicketPDF`, `EventDetail`, `GuestTicket`, `MyTickets`), leaking guest bearer tokens to an attacker domain.

**Verification:** `npm run build` — builds clean. Manual test: `?next=//evil.com` loads SPA normally; `?next=/e/my-event` navigates correctly.

---

### ENV-2026-05 (Low) — Idempotency keys not bound to user/route

**File:** `backend/internal/middleware/idempotency.go:111, 189-194`

**Change:**
```diff
- redisKey := "idem:" + key
+ redisKey := "idem:" + req.Method + ":" + c.Path() + ":" + idemScope(c) + ":" + key
```

New helper:
```go
func idemScope(c echo.Context) string {
    if u, ok := UserFromContext(c); ok {
        return u.ID.String()
    }
    return ""
}
```

**Rationale:** The old Redis key `"idem:<client-key>"` had no route or user scoping. Bodyless endpoints shared the same hash (empty body), so idempotency keys could collide across routes. A `done` entry replayed stored response bodies (including guest `access_token` from registration-create) for 5 minutes to any caller presenting the same key + body hash. While the victim's UUID key is high-entropy (low likelihood), the lack of scoping was a design issue.

After the fix: key is scoped to `idem:<METHOD>:<ROUTE>:<USER_ID>:<CLIENT_KEY>`. Cross-route collision is impossible. Cross-user replay requires knowing both the victim's UUID key and their user ID. Public/guest routes (no authenticated user) have an empty scope segment.

**Verification:** `go test ./internal/middleware/ -run 'TestIdempotency'` — all 6 tests pass (test key format updated).  
**Caveat:** Existing in-flight idempotency keys in Redis use the old flat format and will be treated as new requests (not ideal but safe — they get 201 instead of replayed 200, which is idempotent). The old entries expire naturally (60s in_progress, 5m done).

---

## Build & Test Summary

| Target | Result |
|--------|--------|
| `go build ./...` | ✅ Clean |
| `go test ./internal/server/` | ✅ PASS (1.059s) |
| `go test ./internal/service/` | ✅ PASS (0.767s) |
| `go test ./internal/middleware/ -run TestIdempotency` | ✅ PASS (0.560s) |
| `npm run build` | ✅ Clean (1.84s) |

---

## Not Fixed (deferred)

- **ENV-2026-06 (Info)** — No SECURITY.md, missing HSTS/CSP/X-Frame-Options/Referrer-Policy headers. Requires security policy creation + Caddyfile/SPA config changes.
- **ENV-2026-07 (Info)** — `/healthz` exposes build metadata; CORS is `*`. Minor — HEALTHZ_TOKEN can protect /healthz; wildcard CORS is acceptable for a token-based SPA.
- **ENV-2026-08 (Info)** — DTO URL validator accepts non-http schemes on meeting_url/cover URLs. Neutralized by React sanitization + R2 host check for covers.
- **Orphan R2 object cleanup** — Lifecycle policy or `DeleteObject` on cover nullification not implemented in this round. The `VerifyImage` size check + `ContentLength` cap on presigned URLs close the bucket-fill abuse vector.

---

## Security Posture After Fixes

- **AuthN/AuthZ:** Unchanged — already verified sound in the scan (Clerk JWT, pre-warmed JWKS, race-safe provisioning, scoped SQL, guest linking by verified email).
- **Guest OTP:** Unchanged — already sound (crypto/rand, HMAC-only, constant-time compare, 5-try limit, 10m TTL, single-use marker, per-(event,email) cooldown).
- **Email pipeline:** Unchanged — already sound (html/template auto-escaping, Resend API, transactional outbox, JetStream MsgID idempotency).
- **SQL injection:** No findings in scan — already sqlc/parameterized only.
- **Secrets:** No findings — `.env*` gitignored, CI gitleaks, GitHub Actions secrets.* only.
