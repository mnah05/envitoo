package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/db"
)

const userContextKey = "user"

// UserFromContext returns the authenticated local user stored by the
// LoadUser middleware. The second return is false if the request was
// not authenticated or LoadUser did not run.
func UserFromContext(c echo.Context) (*db.User, bool) {
	u, ok := c.Get(userContextKey).(*db.User)
	return u, ok
}

// Auth verifies the Clerk session JWT from the Authorization header and
// places *clerk.SessionClaims in the request context. It performs no
// database access and no Clerk Backend API calls — JWT verification is
// local crypto against cached JWKS. clerk.SetKey must be called once at
// startup before this middleware is used.
func Auth() echo.MiddlewareFunc {
	return echo.WrapMiddleware(clerkhttp.RequireHeaderAuthorization())
}

// LoadUser provisions the local user for the authenticated request and
// runs first-login onboarding inside a single transaction: insert the
// local user row, link any prior guest registrations matched by email,
// and create a default calendar.
//
// Hot path (returning user): one SELECT by auth_user_id. No Clerk API
// call, no database write, no onboarding.
//
// First login (no local row): calls Clerk user.Get once, then opens a
// transaction and (a) inserts the local user row with ON CONFLICT DO
// NOTHING, (b) links guest registrations, (c) creates a default
// calendar. A concurrent first-login racing on the same auth_user_id is
// handled via ON CONFLICT DO NOTHING — the loser's CreateUser returns
// pgx.ErrNoRows, the transaction rolls back, and the loser re-fetches
// the winning row via GetUserByAuthUserID without running onboarding
// (the winner already did).
//
// The logger is retrieved from the request context via slog.FromContext;
// a logger-injecting middleware must run before this one.
//
// Auth must run before LoadUser so that SessionClaims are available in
// the request context.
func LoadUser(pool *pgxpool.Pool, queries *db.Queries) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			log := LoggerFromContext(ctx)

			claims, ok := clerk.SessionClaimsFromContext(ctx)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			localUser, err := queries.GetUserByAuthUserID(ctx, claims.Subject)
			switch {
			case err == nil:
				linkGuestRegistrations(ctx, queries, log, localUser.ID, localUser.Email)
				c.Set(userContextKey, &localUser)
				return next(c)
			case !errors.Is(err, pgx.ErrNoRows):
				log.ErrorContext(ctx, "lookup local user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			clerkUser, err := user.Get(ctx, claims.Subject)
			if err != nil {
				log.ErrorContext(ctx, "fetch clerk user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			email := primaryEmail(clerkUser)
			if email == "" {
				return echo.NewHTTPError(http.StatusForbidden, "no verified email address")
			}

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
				return refetchAndContinue(ctx, c, queries, claims.Subject, log, next)
			}
			if err != nil {
				log.ErrorContext(ctx, "create local user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			// Link any prior guest registrations (user_id IS NULL) that
			// match the new user's email. Safe w.r.t. the partial unique
			// index idx_registrations_user_event because UNIQUE(event_id,
			// email) guarantees at most one row per event/email.
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
	authUserID string,
	log *slog.Logger,
	next echo.HandlerFunc,
) error {
	localUser, err := queries.GetUserByAuthUserID(ctx, authUserID)
	if err != nil {
		log.ErrorContext(ctx, "refetch local user after race", "clerk_user_id", authUserID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	linkGuestRegistrations(ctx, queries, log, localUser.ID, localUser.Email)
	c.Set(userContextKey, &localUser)
	return next(c)
}

// linkGuestRegistrations links orphaned guest rows to the user and cancels
// guest rows superseded by an existing user registration for the same event.
// Errors are logged but do not fail the request — linking is best-effort on
// the returning-user hot path.
func linkGuestRegistrations(ctx context.Context, queries *db.Queries, log *slog.Logger, userID uuid.UUID, email string) {
	linked, cancelled, err := linkGuestRegistrationsTx(ctx, queries, userID, email)
	if err != nil {
		log.ErrorContext(ctx, "link guest registrations", "user_id", userID, "error", err)
		return
	}
	if linked > 0 || cancelled > 0 {
		log.InfoContext(ctx, "linked guest registrations",
			"user_id", userID, "linked", linked, "cancelled_superseded", cancelled)
	}
}

func linkGuestRegistrationsTx(ctx context.Context, queries *db.Queries, userID uuid.UUID, email string) (linked, cancelled int64, err error) {
	userIDParam := pgtype.UUID{Bytes: userID, Valid: true}

	linked, err = queries.LinkGuestRegistrations(ctx, db.LinkGuestRegistrationsParams{
		Email:  email,
		UserID: userIDParam,
	})
	if err != nil {
		return 0, 0, err
	}

	cancelled, err = queries.CancelSupersededGuestRegistrations(ctx, db.CancelSupersededGuestRegistrationsParams{
		Email:  email,
		UserID: userIDParam,
	})
	if err != nil {
		return linked, 0, err
	}
	return linked, cancelled, nil
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

var (
	slugNonAlnum    = regexp.MustCompile(`[^a-z0-9]+`)
	slugEdgeHyphens = regexp.MustCompile(`^-+|-+$`)
)

// sanitizeSlug lowercases s, collapses runs of non-[a-z0-9] characters
// into a single hyphen, trims leading/trailing hyphens, and truncates
// to 63 characters. Returns "" if s contains no alphanumeric ASCII.
func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = slugEdgeHyphens.ReplaceAllString(s, "")
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
	}
	return s
}

// primaryEmail returns the user's primary email address, or the first
// email listed if no primary is set. Returns "" if the user has no
// email addresses.
func primaryEmail(u *clerk.User) string {
	if u == nil {
		return ""
	}
	if u.PrimaryEmailAddressID != nil {
		for _, e := range u.EmailAddresses {
			if e.ID == *u.PrimaryEmailAddressID {
				return e.EmailAddress
			}
		}
	}
	if len(u.EmailAddresses) > 0 {
		return u.EmailAddresses[0].EmailAddress
	}
	return ""
}
