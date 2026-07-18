---
name: go-backend
description: Use when writing, modifying, or reviewing Go backend code in this project. Covers Echo handlers, pgx/sqlc database patterns, service layer, middleware, DTOs, error handling, caching, and testing conventions for the EnviToo API server.
---

# Go Backend Skill - EnviToo

Backend is Go 1.26, module `github.com/mnah05/envitoo`. Echo v4 HTTP framework, pgx/v5 + sqlc for PostgreSQL, Clerk for auth, Redis for caching. Follow these conventions exactly.

## Architecture layers

```
cmd/server/main.go          -> bootstrap, signal handling
internal/server/            -> HTTP handlers (Server struct)
internal/service/           -> business logic (EventService, RegistrationService, VolunteerService)
internal/db/                -> sqlc-generated query code (DO NOT EDIT)
internal/dto/               -> request/response DTOs + validation
internal/middleware/         -> auth, logging, rate limiting
internal/helper/             -> utilities (slug, token, db_error)
internal/cache/             -> Redis cache-aside with generic GetJSON[T]
internal/config/            -> koanf-based env config
sql/                        -> raw SQL queries (sqlc source of truth)
supabase/migrations/        -> database schema
```

## Server struct and bootstrap

```go
type Server struct {
    pool          *pgxpool.Pool
    queries       *db.Queries
    cache         *cache.Store
    log           *slog.Logger
    validate      *validator.Validate
    events        *service.EventService
    registrations *service.RegistrationService
    volunteers    *service.VolunteerService
}
```

Services are instantiated in `New()`, not injected via interfaces. `Server.Register(e)` wires all routes.

Route groups:
- `public` = `/api` with `PublicRateLimiter`
- `protected` = `/api` with `Auth()`, `LoadUser()`, `AuthRateLimiter()`

## Handler pattern

Every handler follows this exact sequence:

```go
func (s *Server) CreateEvent(c echo.Context) error {
    log := middleware.LoggerFromContext(c.Request().Context())  // 1. logger from ctx

    var req dto.CreateEventRequest                              // 2. bind
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }
    if err := s.validate.Struct(req); err != nil {             // 3. validate
        return validationError(c, err)
    }

    u, ok := middleware.UserFromContext(c)                      // 4. auth user (if protected)
    if !ok {
        return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
    }

    ctx := c.Request().Context()
    event, err := s.events.Create(ctx, u.ID, req)              // 5. call service
    if err != nil {
        log.ErrorContext(ctx, "create event", "organizer_id", u.ID, "error", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
    }

    resp, err := s.events.EventResponseWithCount(ctx, event)   // 6. build response DTO
    if err != nil {
        log.ErrorContext(ctx, "create event response", "error", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
    }

    return c.JSON(http.StatusCreated, resp)                    // 7. JSON response
}
```

### Error mapping in handlers

Use `errors.Is()` chains to map service sentinel errors to HTTP status:

```go
if errors.Is(err, service.ErrEventNotFound) {
    return echo.NewHTTPError(http.StatusNotFound, "event not found")
}
if errors.Is(err, service.ErrEventStaffForbidden) {
    return echo.NewHTTPError(http.StatusForbidden, "insufficient event permissions")
}
log.ErrorContext(ctx, "update event", "id", id, "error", err)
return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
```

Extract repeated error mapping into helper functions that return `*echo.HTTPError` (nil = fall through to 500).

### Path and query params

Path: `uuid.Parse(c.Param("id"))` - always 400 on parse failure.
Query: `parseLimit(c.QueryParam("limit"), 20, 100)`, `parseOffset(c.QueryParam("offset"))` - helper functions.

### Cache-aside reads

```go
resp, err := cache.GetJSON(ctx, s.cache, "GET:/api/events/"+id.String(), cache.EventTTL, func(ctx context.Context) (dto.EventResponse, error) {
    event, err := s.events.GetViewableByID(ctx, id)
    if err != nil {
        return dto.EventResponse{}, err
    }
    return s.events.EventResponseWithCount(ctx, event)
})
```

Cache keys: `"HTTP_METHOD:/path"`. Invalidation: `Delete()` for exact, `DeletePrefix()` for list patterns.

## Service layer

### Service struct

```go
type EventService struct {
    pool    *pgxpool.Pool
    queries *db.Queries
    access  *EventAccessService
    log     *slog.Logger
}
```

Constructor: `NewEventService(pool, queries, access, log) *EventService`

### Transactions

```go
tx, err := s.pool.Begin(ctx)
if err != nil {
    return db.Registration{}, fmt.Errorf("begin tx: %w", err)
}
defer func() { _ = tx.Rollback(ctx) }()

qtx := s.queries.WithTx(tx)
// ... business logic with qtx ...

if err := tx.Commit(ctx); err != nil {
    return db.Registration{}, fmt.Errorf("commit tx: %w", err)
}
```

### Sentinel errors

Package-level `var` in `errors.go`:

```go
var (
    ErrEventNotFound        = errors.New("event not found")
    ErrConflict             = errors.New("registration already exists")
    ErrEventFull            = errors.New("event is full")
)
```

Wrap with context: `fmt.Errorf("event %s not found: %w", id, ErrEventNotFound)`

### Access control

`EventAccessService.ResolveStaffRole(ctx, eventID, actorID)` returns `EventStaffOrganizer`, `EventStaffVolunteerAdmin`, or `EventStaffNone`.

### Pagination

```go
type ListParams struct {
    Limit            int32
    Offset           int32
    IncludeCancelled bool
}
```

Default limit 20, max 100. Offset >= 0.

## sqlc conventions

### Query annotations

```sql
-- name: CreateEvent :one
-- name: ListPublicEvents :many
-- name: DeleteEventVisibility :exec
-- name: LinkGuestRegistrations :execrows
```

### Authorization in SQL

Embed auth checks in WHERE clauses:

```sql
WHERE e.organizer_id = $2
   OR EXISTS (
     SELECT 1 FROM volunteer_applications va
     JOIN volunteer_roles vr ON vr.id = va.role_id
     WHERE va.event_id = e.id AND va.user_id = $2
       AND va.status = 'approved' AND vr.is_admin = true
   )
```

### Pagination with window functions

```sql
SELECT r.*, COUNT(*) OVER()::int4 AS total_count
FROM registrations r WHERE r.user_id = $1
ORDER BY r.created_at DESC LIMIT $2 OFFSET $3;
```

### Idempotent inserts

```sql
INSERT INTO registrations (...) VALUES (...) ON CONFLICT DO NOTHING RETURNING *;
```

Service treats `pgx.ErrNoRows` on these as a conflict (sqlc returns no rows).

### After editing sql/ files

Run `sqlc generate` to regenerate `internal/db/`. Never edit `internal/db/` directly.

## DTO patterns

### Requests

POST/PUT: required fields with `validate:"required,min=1,max=200"`
PATCH: all pointers `*string`, `*int32`, `*bool` for partial updates
JSON tags: snake_case

### Responses

Constructors convert `db.Model -> dto.Response`:

```go
func NewEventResponse(e db.Event) EventResponse { ... }
func NewEventListResponse(events []EventResponse, limit, offset, total int32) EventListResponse { ... }
```

Lists wrap in: `{ events: [...], limit, offset, total }`

Nullable fields: `pgtype.UUID.Valid` -> `*uuid.UUID`, `pgtype.Numeric` -> `*float64`.

### Validation

go-playground/validator with struct tags. Cross-field validations registered in `validator.go`:

```go
func registerEventValidations(v *validator.Validate) {
    v.RegisterStructValidation(validateCreateEventRequest, CreateEventRequest{})
}
```

## DB error helpers (`internal/helper/db_error.go`)

```go
func IsUniqueViolation(err error, constraint string) bool
func IsRegistrationDuplicate(err error) bool
func IsForeignKeyViolation(err error) bool
func IsVolunteerApplicationDuplicate(err error) bool
```

Type-check Postgres errors using `errors.As` with `*pgconn.PgError`.

## Logging

`slog` everywhere. Inject via `middleware.WithLogger(ctx, log)`, retrieve with `middleware.LoggerFromContext(ctx)`. Always `log.ErrorContext(ctx, "operation", "key", value, "error", err)`.

## Testing

- Package-level tests (whitebox): `package service`, `package helper`
- Table-driven tests with named subtests
- `miniredis` for cache tests (in-memory Redis)
- Test helpers: `openTestStore(t)` returns `(*Store, cleanupFunc)`

## Key rules

1. sqlc is the source of truth - write SQL in `sql/`, regenerate with `sqlc generate`
2. Never edit `internal/db/` directly
3. Services use concrete structs, not interfaces
4. Transactions: always `defer tx.Rollback(ctx)` before explicit commit
5. Error wrapping: always `fmt.Errorf("context: %w", err)`
6. Handler logging: always log the error before returning 500
7. Cache invalidation after writes that affect cached reads
