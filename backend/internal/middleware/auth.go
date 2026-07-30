package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/helper"
)

// Context key for storing the local user in Echo's per-request storage.
// Echo's c.Set/c.Get use a map[string]any; a typed constant prevents
// collisions with other middleware that might use the same string key.
const userContextKey = "user"

// UserFromContext returns the authenticated local user stored by the
// LoadUser middleware. The second return is false if the request was
// not authenticated or LoadUser did not run.
func UserFromContext(c echo.Context) (*db.User, bool) {
	u, ok := c.Get(userContextKey).(*db.User)
	return u, ok
}

// Auth is the first gate: it verifies the Clerk session JWT from the
// Authorization header and stores *clerk.SessionClaims in the request
// context. The verification uses local RSA/EC crypto against a cached
// JWKS (JSON Web Key Set) that Clerk publishes — no outbound HTTP call
// to Clerk happens on every request.
//
// clerk.SetKey must be called once at startup before this middleware
// is used.
func Auth() echo.MiddlewareFunc {
	return echo.WrapMiddleware(clerkhttp.RequireHeaderAuthorization())
}

// LoadUser provisions the local user for the authenticated request and
// runs first-login onboarding inside a single transaction: insert the
// local user row, link any prior guest registrations matched by email,
// and create a default calendar.
//
// --- Design: hot path vs cold path ---
//
// Returning-user hot path (most requests): one SELECT by auth_user_id.
// No Clerk API call, no database write, no onboarding.
//
// First-login cold path: calls Clerk user.Get once, then opens a
// database transaction that (a) inserts the local user row with ON
// CONFLICT DO NOTHING, (b) links guest registrations, (c) creates a
// default calendar.
//
// If two first-login requests race on the same Clerk user (rare but
// possible with retries or client-side double-submit), ON CONFLICT DO
// NOTHING ensures only one succeeds. The loser's CreateUser returns
// pgx.ErrNoRows, the transaction rolls back (via defer), and control
// flows to refetchAndContinue which re-fetches the winning row without
// re-running onboarding.
//
// The logger is retrieved from the request context via slog.FromContext;
// a logger-injecting middleware must run before this one.
//
// GuestRegistrationInvalidator is called after guest registrations are
// linked or cancelled, so the cache layer can invalidate stale entries
// for the affected email and user.
type GuestRegistrationInvalidator func(ctx context.Context, userID uuid.UUID, email string)

// Auth must run before LoadUser so that SessionClaims are available in
// the request context.
func LoadUser(pool *pgxpool.Pool, queries *db.Queries, cacheStore *cache.Store, onGuestRegsChanged GuestRegistrationInvalidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			log := LoggerFromContext(ctx)

			// clerk.SessionClaimsFromContext reads the verified JWT claims that
			// the Auth middleware placed in the request context earlier.
			claims, ok := clerk.SessionClaimsFromContext(ctx)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			// Hot path: SELECT from the local users table, cached in Redis.
			// No Clerk API call.
			// claims.Subject is the Clerk auth_user_id (e.g. "user_2abc...").
			localUser, err := cache.GetJSON(ctx, cacheStore, "user:auth:"+claims.Subject, 2*time.Minute, func(ctx context.Context) (db.User, error) {
				return queries.GetUserByAuthUserID(ctx, claims.Subject)
			})
			switch {
			case err == nil:
				// Normalize for guest-link matching (guest emails are stored lowercase).
				linkGuestRegistrations(ctx, queries, log, localUser.ID, helper.NormalizeEmail(localUser.Email), onGuestRegsChanged)
				c.Set(userContextKey, &localUser)
				return next(c)
			case !errors.Is(err, pgx.ErrNoRows):
				log.ErrorContext(ctx, "lookup local user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			// Cold path: first login for this Clerk user — no local row exists.
			// Fetch user profile from Clerk's Backend API (one outbound HTTP call).
			clerkUser, err := user.Get(ctx, claims.Subject)
			if err != nil {
				log.ErrorContext(ctx, "fetch clerk user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			// We require at least one verified email address. Unverified
			// or missing emails return 403 so the frontend can prompt the
			// user to verify their email in Clerk's account portal.
			email := helper.NormalizeEmail(primaryEmail(clerkUser))
			if email == "" {
				return echo.NewHTTPError(http.StatusForbidden, "no verified email address")
			}

			// Everything from here through Commit must succeed atomically.
			tx, err := pool.Begin(ctx)
			if err != nil {
				log.ErrorContext(ctx, "begin onboarding tx", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
			defer func() {
				if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					log.ErrorContext(ctx, "rollback onboarding tx", "clerk_user_id", claims.Subject, "error", err)
				}
			}()

			qtx := queries.WithTx(tx)

			created, err := qtx.CreateUser(ctx, db.CreateUserParams{
				AuthUserID: claims.Subject,
				Email:      email,
				Username:   clerkUser.Username,
				AvatarUrl:  clerkUser.ImageURL,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				// Race loser: another request (or webhook) inserted the
				// user first. Roll back (defer) and re-fetch the winning
				// row; skip onboarding — the winner already did it.
				return refetchAndContinue(ctx, c, queries, cacheStore, claims.Subject, log, onGuestRegsChanged, next)
			}
			if err != nil {
				log.ErrorContext(ctx, "create local user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			// Link any prior guest registrations (user_id IS NULL) that
			// match the new user's email. Safe w.r.t. the partial unique
			// index idx_registrations_user_event (active statuses only):
			// LinkGuestRegistrations skips events where the user already
			// has an active (pending/approved) row; cancelled/rejected
			// user rows do not block linking. idx_registrations_event_email_active
			// allows at most one active guest row per event/email.
			linked, cancelled, err := linkGuestRegistrationsTx(ctx, qtx, created.ID, email)
			if err != nil {
				log.ErrorContext(ctx, "link guest registrations", "user_id", created.ID, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			// Create the user's default calendar. ON CONFLICT DO NOTHING
			// turns any unique violation into pgx.ErrNoRows. On the
			// winner path a user_id conflict is impossible, so
			// ErrNoRows means the slug collided with another user's
			// calendar — retry once with the user's UUID as the slug,
			// which is globally unique.
			slug, name := defaultCalendarSlugName(created)
			_, err = qtx.CreateCalendar(ctx, db.CreateCalendarParams{
				UserID: created.ID,
				Slug:   slug,
				Name:   name,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				slug, name = created.ID.String(), "My Calendar"
				_, err = qtx.CreateCalendar(ctx, db.CreateCalendarParams{
					UserID: created.ID,
					Slug:   slug,
					Name:   name,
				})
			}
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.ErrorContext(ctx, "create default calendar", "user_id", created.ID, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
			if errors.Is(err, pgx.ErrNoRows) {
				if _, fetchErr := qtx.GetCalendarByUserID(ctx, created.ID); fetchErr != nil {
					if errors.Is(fetchErr, pgx.ErrNoRows) {
						log.ErrorContext(ctx, "default calendar missing after onboarding", "user_id", created.ID)
						return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
					}
					log.ErrorContext(ctx, "fetch default calendar after onboarding", "user_id", created.ID, "error", fetchErr)
					return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
				}
			}

			if err := tx.Commit(ctx); err != nil {
				log.ErrorContext(ctx, "commit onboarding tx", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			if linked > 0 || cancelled > 0 {
				log.InfoContext(ctx, "linked guest registrations on onboarding",
					"user_id", created.ID, "linked", linked, "cancelled_superseded", cancelled)
				if onGuestRegsChanged != nil {
					onGuestRegsChanged(ctx, created.ID, email)
				}
			}
			log.InfoContext(ctx, "user onboarded", "user_id", created.ID, "calendar_slug", slug)

			c.Set(userContextKey, &created)
			return next(c)
		}
	}
}

// refetchAndContinue re-fetches the local user by Clerk auth_user_id
// after a CreateUser race loss (ON CONFLICT DO NOTHING returned no
// rows), stores it in the Echo context, and invokes the next handler.
// It uses the pool-bound queries, not the transaction, since the
// transaction is being rolled back by the caller's defer. Onboarding is
// not run — the race winner already performed it.
func refetchAndContinue(
	ctx context.Context,
	c echo.Context,
	queries *db.Queries,
	cacheStore *cache.Store,
	authUserID string,
	log *slog.Logger,
	onGuestRegsChanged GuestRegistrationInvalidator,
	next echo.HandlerFunc,
) error {
	localUser, err := cache.GetJSON(ctx, cacheStore, "user:auth:"+authUserID, 2*time.Minute, func(ctx context.Context) (db.User, error) {
		return queries.GetUserByAuthUserID(ctx, authUserID)
	})
	if err != nil {
		log.ErrorContext(ctx, "refetch local user after race", "clerk_user_id", authUserID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	linkGuestRegistrations(ctx, queries, log, localUser.ID, localUser.Email, onGuestRegsChanged)
	c.Set(userContextKey, &localUser)
	return next(c)
}

// linkGuestRegistrations links orphaned guest rows to the user and cancels
// guest rows superseded by an existing user registration for the same event.
// Errors are logged but do not fail the request — linking is best-effort on
// the returning-user hot path.
//
// On the hot path, >99% of users have no orphaned guest registrations, so we
// check first with a cheap EXISTS query and skip entirely when there's nothing
// to do. This avoids two UPDATE round-trips on nearly every authenticated
// request.
func linkGuestRegistrations(ctx context.Context, queries *db.Queries, log *slog.Logger, userID uuid.UUID, email string, onGuestRegsChanged GuestRegistrationInvalidator) {
	hasOrphans, err := queries.CheckOrphanedGuestRegistrations(ctx, email)
	if err != nil {
		log.ErrorContext(ctx, "check orphaned guest registrations", "user_id", userID, "error", err)
		return
	}
	if !hasOrphans {
		return
	}

	linked, cancelled, err := linkGuestRegistrationsTx(ctx, queries, userID, email)
	if err != nil {
		log.ErrorContext(ctx, "link guest registrations", "user_id", userID, "error", err)
		return
	}
	if linked > 0 || cancelled > 0 {
		log.InfoContext(ctx, "linked guest registrations",
			"user_id", userID, "linked", linked, "cancelled_superseded", cancelled)
		if onGuestRegsChanged != nil {
			onGuestRegsChanged(ctx, userID, email)
		}
	}
}

func linkGuestRegistrationsTx(ctx context.Context, queries *db.Queries, userID uuid.UUID, email string) (linked, cancelled int64, err error) {
	userIDParam := pgtype.UUID{Bytes: userID, Valid: true}
	return queries.LinkGuestRegistrationsCombined(ctx, email, userIDParam)
}

// defaultCalendarSlugName derives a slug and display name for the
// user's default calendar from their username. If the username is empty
// or sanitizes to an empty string, it falls back to the user's UUID as
// the slug and "My Calendar" as the name. A slug collision with another
// user's calendar is handled by the caller, which retries once with the
// UUID slug.
func defaultCalendarSlugName(u db.User) (slug, name string) {
	if u.Username == nil || *u.Username == "" {
		return u.ID.String(), "My Calendar"
	}
	s := sanitizeSlug(*u.Username)
	if s == "" {
		return u.ID.String(), "My Calendar"
	}
	return s, *u.Username + "'s Calendar"
}

// Pre-compiled regexes used by sanitizeSlug. Compiling once at package
// init avoids re-parsing on every call and is goroutine-safe (Regexp is
// immutable after Compile).
// The first replaces non-alphanumeric chars with hyphens; the second
// strips leading/trailing hyphens.
var (
	slugNonAlnum    = regexp.MustCompile(`[^a-z0-9]+`) // runs of non-alphanumeric chars
	slugEdgeHyphens = regexp.MustCompile(`^-+|-+$`)    // leading or trailing hyphens
)

// sanitizeSlug converts a free-text string into a URL-safe slug suitable
// for use in paths like /calendar/<slug>. The algorithm:
//  1. Lowercase the input.
//  2. Replace every run of one or more non-alphanumeric characters with a
//     single hyphen (e.g. "John's Faves!!" → "john-s-faves-").
//  3. Strip leading/trailing hyphens (→ "john-s-faves").
//  4. Truncate to 63 characters (a common DB/URL length limit), removing
//     any trailing hyphen the truncation might leave.
//
// Returns "" when the input has no alphanumeric characters at all.
func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = slugEdgeHyphens.ReplaceAllString(s, "")
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
	}
	return s
}

// primaryEmail returns a verified email for the user. Prefers the primary
// address when it is verified; otherwise the first verified address.
// Returns "" when no verified email exists (caller maps this to 403).
func primaryEmail(u *clerk.User) string {
	if u == nil {
		return ""
	}
	if u.PrimaryEmailAddressID != nil {
		for _, e := range u.EmailAddresses {
			if e != nil && e.ID == *u.PrimaryEmailAddressID && emailVerified(e) {
				return e.EmailAddress
			}
		}
	}
	for _, e := range u.EmailAddresses {
		if e != nil && emailVerified(e) {
			return e.EmailAddress
		}
	}
	return ""
}

func emailVerified(e *clerk.EmailAddress) bool {
	return e.Verification != nil && e.Verification.Status == "verified"
}
