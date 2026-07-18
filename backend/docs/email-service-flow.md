# Email Outbox Implementation Report

## Files changed (13 files, +917/-90)

- `sql/registration.sql` — 3 queries modified, 1 new query
- `sql/event.sql` — 1 query modified
- `sql/user.sql` — 1 new query
- `internal/db/*.sql.go` — 4 regenerated files (sqlc)
- `internal/service/email_templates.go` — **new file**
- `internal/service/registration.go` — 8 methods converted
- `internal/service/volunteer.go` — 4 methods converted
- `internal/service/event.go` — 1 method converted
- `internal/service/guest_otp.go` — removed duplicate constants
- `cmd/worker/drainer.go` — added volunteer_status_change template

---

## 1. SQL changes (registration.sql)

### CancelRegistration — idempotency guard added
```
- AND user_id = $2
+ AND user_id = $2 AND status != 'cancelled'
```
Re-cancel returns zero rows → `pgx.ErrNoRows` → `ErrRegistrationNotFound` → no email.

### CancelGuestRegistration — idempotency guard added
```
- AND user_id IS NULL
+ AND user_id IS NULL AND status != 'cancelled'
```
Same as above: re-cancel returns 404, no email.

### UpdateRegistrationStatus — expected-status guard added
```
- AND e.organizer_id = $2
+ AND e.organizer_id = $2 AND r.status = $4
```
New `$4` parameter (`Status_2` in generated Go) is the status the caller read before the update. If a concurrent change already moved it, zero rows → `ErrConflict`.

### GetRegistrationByIDForUpdate — new query
```sql
SELECT r.* FROM registrations r WHERE r.id = $1 FOR UPDATE;
```
Used by `CheckIn` to serialize concurrent check-in attempts within a transaction.

---

## 2. SQL changes (event.sql)

### CancelEvent — idempotency guard added
```
- AND organizer_id = $2 RETURNING *;
+ AND organizer_id = $2 AND status != 'cancelled' RETURNING *;
```

---

## 3. SQL changes (user.sql)

### GetUserByID — new query
```sql
SELECT * FROM users WHERE id = $1;
```
Needed by `EventService.Cancel` to fetch the organizer's email for the cancellation outbox row.

---

## 4. New file: internal/service/email_templates.go

Single source of truth for all email template/topic constants and template data structs.

**9 template name constants**: `EmailTemplateOTP`, `EmailTemplateTicketConfirm`, `EmailTemplateTicketCancel`, `EmailTemplateTicketStatusChange`, `EmailTemplateApplicationConfirm`, `EmailTemplateApplicationCancel`, `EmailTemplateVolunteerStatusChange`, `EmailTemplateCheckIn`, `EmailTemplateEventCancellation`

**9 NATS topic constants**: `EmailTopicOTP` → `"emails.otp"`, `EmailTopicTicketConfirm` → `"emails.ticket.confirmed"`, etc.

**7 template data structs** (with json tags):
- `ticketConfirmData` — Name, EventTitle, EventID, AccessToken
- `ticketCancelData` — Name, EventTitle, EventID
- `ticketStatusChangeData` — Name, EventTitle, EventID, NewStatus
- `applicationConfirmData` — Name, EventTitle, EventID, RoleName
- `applicationCancelData` — Name, EventTitle, EventID
- `volunteerStatusChangeData` — Name, EventTitle, EventID, NewStatus, RoleName
- `checkInData` — Name, EventTitle, EventID
- `eventCancellationData` — Title, EventID

---

## 5. cmd/worker/drainer.go

Added `TemplateVolunteerStatusChange` (`"volunteer_status_change"`) to the existing template constants and `topicForTemplate` map pointing to `"emails.volunteer.status_changed"`.

---

## 6. internal/service/registration.go — 8 methods

All outbox inserts follow the same pattern:
```
marshal templateData → qtx.CreateEmailOutbox → tx.Commit
```
With the comment `// Outbox insert is transactional — failure rolls back the entire operation.`

### CreateGuest (already in TX)
Before `tx.Commit`: marshal `ticketConfirmData` using `event.Title` (already loaded), `CreateEmailOutbox` with `EmailTemplateTicketConfirm` / `EmailTopicTicketConfirm`.

### Create (already in TX)
Same as CreateGuest: `ticketConfirmData`, event title already available.

### CancelGuest (was not in TX)
Wrapped: `pool.Begin` → `qtx.CancelGuestRegistration` → `qtx.GetEventByID` → marshal `ticketCancelData` → `CreateEmailOutbox` → `tx.Commit`.

### Cancel (was not in TX)
Wrapped: `pool.Begin` → `qtx.CancelRegistration` → `qtx.GetEventByID` → marshal `ticketCancelData` → `CreateEmailOutbox` → `tx.Commit`.

### Approve (already in TX)
- Race guard: `ApproveRegistration` returns `pgx.ErrNoRows` → now returns **`ErrConflict`** (was `ErrRegistrationNotFound`)
- Before commit: fetch event title via `qtx.GetEventByIDForUpdate` (already loaded), marshal `ticketStatusChangeData`, `CreateEmailOutbox`.

### Reject (was not in TX)
Wrapped: `pool.Begin` → `qtx.RejectRegistration` → `qtx.GetEventByID` → marshal `ticketStatusChangeData` → `CreateEmailOutbox` → `tx.Commit`.
- Race guard: `pgx.ErrNoRows` → **`ErrConflict`** (was `ErrRegistrationNotFound`)

### UpdateStatus (already in TX)
- Passes `reg.Status` as new `Status_2` param (expected current status)
- Race guard: `pgx.ErrNoRows` → **`ErrConflict`** (was `ErrRegistrationNotFound`)
- Before commit: fetch event title via `qtx.GetEventByID`, marshal `ticketStatusChangeData`, `CreateEmailOutbox`.

### CheckIn (was not in TX)
Wrapped: `pool.Begin` → `qtx.GetRegistrationByIDForUpdate` (lock row) → `qtx.CheckInRegistration` → on success: `qtx.GetEventByID` → marshal `checkInData` → `CreateEmailOutbox` → `tx.Commit`.
- Idempotent: already-checked-in skips outbox, commits and returns existing.
- All fallback queries use `qtx` instead of `s.queries`.

### CheckInByEventToken (was not in TX)
Wrapped: `pool.Begin` → `qtx.CheckInRegistrationByEventToken` → on success: `qtx.GetEventByID` → marshal `checkInData` → `CreateEmailOutbox` → `tx.Commit`.
- Same idempotent handling and qtx fallback as CheckIn.

---

## 7. internal/service/volunteer.go — 4 methods

### Apply (already in TX)
Before commit: `qtx.GetEventByID` → marshal `applicationConfirmData` with `role.Name` → `CreateEmailOutbox`.

### Approve (already in TX)
- Fixed `volunteerModerationError` call to use `qtx` (was `s.queries`)
- Before commit: `qtx.GetEventByID` → marshal `volunteerStatusChangeData` with `role.Name` → `CreateEmailOutbox`.

### Cancel (was not in TX)
Wrapped: `pool.Begin` → `qtx.CancelVolunteerApplication` → `qtx.GetEventByID` → `qtx.GetVolunteerRoleByID` → marshal `applicationCancelData` → `CreateEmailOutbox` → `tx.Commit`.
- Uses `qtx` in `volunteerCancelError` (was `s.queries`).

### Reject (was not in TX)
Wrapped: `pool.Begin` → `qtx.RejectVolunteerApplication` → `qtx.GetEventByID` → `qtx.GetVolunteerRoleByID` → marshal `volunteerStatusChangeData` → `CreateEmailOutbox` → `tx.Commit`.
- Uses `qtx` in `volunteerModerationError` (was `s.queries`).

---

## 8. internal/service/event.go — 1 method

### Cancel (was not in TX)
Wrapped: `pool.Begin` → `qtx.CancelEvent` → `qtx.GetUserByID` (for organizer email) → marshal `eventCancellationData` → `CreateEmailOutbox` → `tx.Commit`.

---

## 9. internal/service/guest_otp.go

Removed duplicate constants `emailTemplateOTP` and `emailTopicOTP`. The `RequestGuestOTP` method now uses `EmailTemplateOTP` and `EmailTopicOTP` from `email_templates.go`.

---

## Coverage

| Template | Topic | Producer methods | Status |
|----------|-------|-----------------|--------|
| `otp` | `emails.otp` | `RequestGuestOTP` | Already done |
| `ticket_confirm` | `emails.ticket.confirmed` | `CreateGuest`, `Create` | Done |
| `ticket_cancel` | `emails.ticket.cancelled` | `CancelGuest`, `Cancel` | Done |
| `ticket_status_change` | `emails.ticket.status_changed` | `Approve`, `Reject`, `UpdateStatus` | Done |
| `application_confirm` | `emails.application.confirmed` | `Apply` | Done |
| `application_cancel` | `emails.application.cancelled` | `Cancel` (volunteer) | Done |
| `volunteer_status_change` | `emails.volunteer.status_changed` | `Approve`, `Reject` (volunteer) | Done |
| `check_in` | `emails.checkin` | `CheckIn`, `CheckInByEventToken` | Done |
| `event_cancellation` | `emails.event.cancelled` | `Cancel` (event) | Done |

**15 outbox insert sites across 13 service methods, covering all 9 templates.**
