package server

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/user"
)

// ProvisionMe handles POST /api/me: it returns the authenticated user's
// profile, running first-login provisioning (insert users row, link
// guest registrations, create default calendar) when no local row
// exists. This is the explicit onboarding entry point — the frontend
// calls it once from /welcome after signup. Idempotent: repeat calls
// return the existing user.
//
// Unlike GET /api/me this route does not run LoadUser; user resolution
// happens here via user.FindOrProvision so onboarding has a single,
// explicit trigger.
func (s *Server) ProvisionMe(c echo.Context) error {
	ctx := c.Request().Context()
	log := middleware.LoggerFromContext(ctx)

	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	u, err := user.FindOrProvision(ctx, s.pool, s.queries, s.cache, claims.Subject, log, s.forgetRegistrationsByEmail)
	if err != nil {
		return middleware.ProvisionHTTPError(err)
	}
	return c.JSON(http.StatusOK, dto.NewUserResponse(u))
}
