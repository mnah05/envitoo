// Package user contains local user provisioning: the first-login flow
// that inserts the users row, links any prior guest registrations
// matched by email, and creates the default calendar in a single
// transaction.
//
// Provisioning has exactly two entry points:
//   - POST /api/me (user.FindOrProvision) — the explicit onboarding
//     endpoint the frontend calls from /welcome after signup.
//   - LoadUser's cold path (user.Provision) — kept for backwards
//     compatibility until clients no longer rely on implicit
//     provisioning; it delegates here so the logic lives in one place.
package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/helper"
)

const (
	// cacheKeyPrefix prefixes the per-user cache key read by LoadUser's
	// hot path. Keep in sync with the key used there.
	cacheKeyPrefix = "user:auth:"
	// cacheTTL bounds how long a resolved user row is cached.
	cacheTTL = 2 * time.Minute
)

// GuestRegistrationInvalidator is called after guest registrations are
// linked or cancelled, so the cache layer can invalidate stale entries
// for the affected email and user.
type GuestRegistrationInvalidator func(ctx context.Context, userID uuid.UUID, email string)

// Provisioning failures that map to distinct HTTP responses at the
// transport layer (see middleware.ProvisionHTTPError).
var (
	// ErrNoVerifiedEmail means the Clerk user has no verified email
	// address; the client must verify one in Clerk's account portal.
	ErrNoVerifiedEmail = errors.New("user: no verified email address")
	// ErrClerkLookup means the Clerk Backend API call failed.
	ErrClerkLookup = errors.New("user: clerk lookup failed")
)

// FindOrProvision returns the local user for a Clerk user ID, running
// first-login provisioning when no local row exists. It mirrors the
// LoadUser hot path (cache → DB → guest-reg linking) and falls back to
// Provision on a cache-and-DB miss. Safe to call repeatedly — repeat
// calls return the existing user without re-running onboarding.
func FindOrProvision(
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *db.Queries,
	cacheStore *cache.Store,
	clerkUserID string,
	log *slog.Logger,
	onGuestRegsChanged GuestRegistrationInvalidator,
) (*db.User, error) {
	localUser, err := cache.GetJSON(ctx, cacheStore, cacheKeyPrefix+clerkUserID, cacheTTL, func(ctx context.Context) (db.User, error) {
		u, err := queries.GetUserByAuthUserID(ctx, clerkUserID)
		if err == nil {
			LinkGuestRegistrations(ctx, queries, log, u.ID, helper.NormalizeEmail(u.Email), onGuestRegsChanged)
		}
		return u, err
	})
	switch {
	case err == nil:
		return &localUser, nil
	case !errors.Is(err, pgx.ErrNoRows):
		log.ErrorContext(ctx, "lookup local user", "clerk_user_id", clerkUserID, "error", err)
		return nil, err
	}
	return Provision(ctx, pool, queries, cacheStore, clerkUserID, log, onGuestRegsChanged)
}

// Provision runs first-login onboarding for a Clerk user that has no
// local row: it fetches the profile from Clerk's Backend API (one
// outbound HTTP call), then opens a single transaction that (a) inserts
// the users row with ON CONFLICT DO NOTHING, (b) links guest
// registrations, and (c) creates the default calendar.
//
// If two provisioning requests race on the same Clerk user, ON CONFLICT
// DO NOTHING ensures only one wins. The loser's CreateUser returns
// pgx.ErrNoRows, the transaction rolls back (via defer), and refetch
// returns the winning row without re-running onboarding.
//
// On success the fresh user is written to the cache so the next LoadUser
// hot-path lookup is a hit.
func Provision(
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *db.Queries,
	cacheStore *cache.Store,
	clerkUserID string,
	log *slog.Logger,
	onGuestRegsChanged GuestRegistrationInvalidator,
) (*db.User, error) {
	clerkUser, err := clerkuser.Get(ctx, clerkUserID)
	if err != nil {
		log.ErrorContext(ctx, "fetch clerk user", "clerk_user_id", clerkUserID, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrClerkLookup, err)
	}

	// We require at least one verified email address. Unverified or
	// missing emails surface as ErrNoVerifiedEmail so the transport
	// layer can return 403 and the frontend can prompt the user to
	// verify their email in Clerk's account portal.
	email := helper.NormalizeEmail(primaryEmail(clerkUser))
	if email == "" {
		return nil, ErrNoVerifiedEmail
	}

	// Everything from here through Commit must succeed atomically.
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.ErrorContext(ctx, "begin onboarding tx", "clerk_user_id", clerkUserID, "error", err)
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.ErrorContext(ctx, "rollback onboarding tx", "clerk_user_id", clerkUserID, "error", err)
		}
	}()

	qtx := queries.WithTx(tx)

	created, err := qtx.CreateUser(ctx, db.CreateUserParams{
		AuthUserID: clerkUserID,
		Email:      email,
		Username:   clerkUser.Username,
		AvatarUrl:  clerkUser.ImageURL,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Race loser: another request (or webhook) inserted the user
		// first. Roll back (defer) and re-fetch the winning row; skip
		// onboarding — the winner already did it.
		return refetch(ctx, queries, cacheStore, clerkUserID, log, onGuestRegsChanged)
	}
	if err != nil {
		log.ErrorContext(ctx, "create local user", "clerk_user_id", clerkUserID, "error", err)
		return nil, err
	}

	// Link any prior guest registrations (user_id IS NULL) that match
	// the new user's email. Safe w.r.t. the partial unique index
	// idx_registrations_user_event (active statuses only):
	// LinkGuestRegistrations skips events where the user already has an
	// active (pending/approved) row; cancelled/rejected user rows do not
	// block linking. idx_registrations_event_email_active allows at most
	// one active guest row per event/email.
	linked, cancelled, err := linkGuestRegistrationsCombined(ctx, qtx, created.ID, email)
	if err != nil {
		log.ErrorContext(ctx, "link guest registrations", "user_id", created.ID, "error", err)
		return nil, err
	}

	// Create the user's default calendar. ON CONFLICT DO NOTHING turns
	// any unique violation into pgx.ErrNoRows. On the winner path a
	// user_id conflict is impossible, so ErrNoRows means the slug
	// collided with another user's calendar — retry once with the
	// user's UUID as the slug, which is globally unique.
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
		return nil, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if _, fetchErr := qtx.GetCalendarByUserID(ctx, created.ID); fetchErr != nil {
			if errors.Is(fetchErr, pgx.ErrNoRows) {
				log.ErrorContext(ctx, "default calendar missing after onboarding", "user_id", created.ID)
				return nil, fetchErr
			}
			log.ErrorContext(ctx, "fetch default calendar after onboarding", "user_id", created.ID, "error", fetchErr)
			return nil, fetchErr
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.ErrorContext(ctx, "commit onboarding tx", "clerk_user_id", clerkUserID, "error", err)
		return nil, err
	}

	if linked > 0 || cancelled > 0 {
		log.InfoContext(ctx, "linked guest registrations on onboarding",
			"user_id", created.ID, "linked", linked, "cancelled_superseded", cancelled)
		if onGuestRegsChanged != nil {
			onGuestRegsChanged(ctx, created.ID, email)
		}
	}
	log.InfoContext(ctx, "user onboarded", "user_id", created.ID, "calendar_slug", slug)

	warmCache(ctx, cacheStore, clerkUserID, created)
	return &created, nil
}

// refetch re-reads the local user after a CreateUser race loss (ON
// CONFLICT DO NOTHING returned no rows). It uses pool-bound queries, not
// the transaction being rolled back by the caller's defer. Onboarding is
// not re-run — the race winner already performed it.
func refetch(
	ctx context.Context,
	queries *db.Queries,
	cacheStore *cache.Store,
	clerkUserID string,
	log *slog.Logger,
	onGuestRegsChanged GuestRegistrationInvalidator,
) (*db.User, error) {
	localUser, err := cache.GetJSON(ctx, cacheStore, cacheKeyPrefix+clerkUserID, cacheTTL, func(ctx context.Context) (db.User, error) {
		u, err := queries.GetUserByAuthUserID(ctx, clerkUserID)
		if err == nil {
			LinkGuestRegistrations(ctx, queries, log, u.ID, u.Email, onGuestRegsChanged)
		}
		return u, err
	})
	if err != nil {
		log.ErrorContext(ctx, "refetch local user after race", "clerk_user_id", clerkUserID, "error", err)
		return nil, err
	}
	return &localUser, nil
}

// warmCache stores a freshly provisioned user under the same key the
// LoadUser hot path reads, so the next request is a cache hit. Failures
// are soft — Store.Set already swallows Redis errors, and a missing
// entry just means the next lookup hits the database.
func warmCache(ctx context.Context, cacheStore *cache.Store, clerkUserID string, u db.User) {
	data, err := json.Marshal(u)
	if err != nil {
		return
	}
	_ = cacheStore.Set(ctx, cacheKeyPrefix+clerkUserID, data, cacheTTL)
}

// LinkGuestRegistrations links orphaned guest rows to the user and
// cancels guest rows superseded by an existing user registration for the
// same event. Errors are logged but do not fail the request — linking is
// best-effort.
//
// >99% of users have no orphaned guest registrations, so we check first
// with a cheap EXISTS query and skip entirely when there's nothing to
// do. This avoids two UPDATE round-trips on nearly every database-load
// path.
//
// This function is only called when the user record is loaded from the
// database (cache MISS), not on every request. The cache TTL bounds how
// stale the "no orphans" state can be.
func LinkGuestRegistrations(ctx context.Context, queries *db.Queries, log *slog.Logger, userID uuid.UUID, email string, onGuestRegsChanged GuestRegistrationInvalidator) {
	hasOrphans, err := queries.CheckOrphanedGuestRegistrations(ctx, email)
	if err != nil {
		log.ErrorContext(ctx, "check orphaned guest registrations", "user_id", userID, "error", err)
		return
	}
	if !hasOrphans {
		return
	}

	linked, cancelled, err := linkGuestRegistrationsCombined(ctx, queries, userID, email)
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

func linkGuestRegistrationsCombined(ctx context.Context, queries *db.Queries, userID uuid.UUID, email string) (linked, cancelled int64, err error) {
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

// primaryEmail returns a verified email for the user. Prefers the
// primary address when it is verified; otherwise the first verified
// address. Returns "" when no verified email exists (caller maps this
// to ErrNoVerifiedEmail).
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
