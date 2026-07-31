package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/helper"
	"github.com/mnah05/envitoo/internal/user"
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
func Auth(jwksClient *jwks.Client) echo.MiddlewareFunc {
	opts := []clerkhttp.AuthorizationOption{}
	if jwksClient != nil {
		opts = append(opts, clerkhttp.JWKSClient(jwksClient))
	}
	return echo.WrapMiddleware(clerkhttp.RequireHeaderAuthorization(opts...))
}

// GuestRegistrationInvalidator is called after guest registrations are
// linked or cancelled, so the cache layer can invalidate stale entries
// for the affected email and user.
type GuestRegistrationInvalidator = user.GuestRegistrationInvalidator

// LoadUser resolves the local user for the authenticated request.
//
// --- Design: hot path vs cold path ---
//
// Returning-user hot path (most requests): one SELECT by auth_user_id,
// cached in Redis. No Clerk API call, no database write, no onboarding.
//
// First-login cold path: delegates to user.Provision, the shared
// onboarding flow also used by POST /api/me. Keeping it here preserves
// implicit provisioning for clients that have not switched to the
// explicit POST /api/me call yet; it will be removed once they have.
//
// The logger is retrieved from the request context via slog.FromContext;
// a logger-injecting middleware must run before this one.
//
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
				u, err := queries.GetUserByAuthUserID(ctx, claims.Subject)
				if err == nil {
					user.LinkGuestRegistrations(ctx, queries, log, u.ID, helper.NormalizeEmail(u.Email), onGuestRegsChanged)
				}
				return u, err
			})
			switch {
			case err == nil:
				c.Set(userContextKey, &localUser)
				return next(c)
			case !errors.Is(err, pgx.ErrNoRows):
				log.ErrorContext(ctx, "lookup local user", "clerk_user_id", claims.Subject, "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}

			// Cold path: first login for this Clerk user — no local row
			// exists. Provision inserts the row, links guest
			// registrations, and creates the default calendar.
			provisioned, err := user.Provision(ctx, pool, queries, cacheStore, claims.Subject, log, onGuestRegsChanged)
			if err != nil {
				return ProvisionHTTPError(err)
			}

			c.Set(userContextKey, provisioned)
			return next(c)
		}
	}
}

// ProvisionHTTPError maps provisioning failures to HTTP responses:
// unverified email → 403 (prompt email verification), Clerk API
// failure → 401, anything else → 500. Shared by LoadUser's cold path
// and the POST /api/me handler.
func ProvisionHTTPError(err error) error {
	switch {
	case errors.Is(err, user.ErrNoVerifiedEmail):
		return echo.NewHTTPError(http.StatusForbidden, "no verified email address")
	case errors.Is(err, user.ErrClerkLookup):
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
}
