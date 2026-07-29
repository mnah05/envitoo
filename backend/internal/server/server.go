package server

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/mnah05/envitoo/internal/cache"
	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/middleware"
	"github.com/mnah05/envitoo/internal/service"
)

// Server holds the HTTP handlers and their dependencies.
type Server struct {
	pool          *pgxpool.Pool
	queries       *db.Queries
	cache         *cache.Store
	log           *slog.Logger
	validate      *validator.Validate
	events        *service.EventService
	registrations *service.RegistrationService
	volunteers    *service.VolunteerService
	healthToken   string
}

// New creates a Server with all services initialized.
// otpSecret is the HMAC key for guest email OTP (OTP_SECRET env).
// healthToken, when non-empty, must be supplied as ?pass= on /healthz.
func New(pool *pgxpool.Pool, queries *db.Queries, c *cache.Store, otpSecret, healthToken string, log *slog.Logger) *Server {
	validate := dto.NewValidator()
	access := service.NewEventAccessService(queries)
	return &Server{
		pool:          pool,
		queries:       queries,
		cache:         c,
		log:           log,
		validate:      validate,
		events:        service.NewEventService(pool, queries, access, log),
		registrations: service.NewRegistrationService(pool, queries, access, c, otpSecret, log),
		volunteers:    service.NewVolunteerService(pool, queries, access, log),
		healthToken:   healthToken,
	}
}

// Register wires all routes onto the given Echo instance.
// ctx is used to stop rate-limiter cleanup goroutines on shutdown.
func (s *Server) Register(ctx context.Context, e *echo.Echo) {
	e.IPExtractor = echo.ExtractIPFromXFFHeader(echo.TrustLoopback(true))
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.RequestID())
	e.Use(middleware.RequestLogger(s.log))

	e.Use(echoMiddleware.BodyLimit("1MB"))
	e.Use(echoMiddleware.TimeoutWithConfig(echoMiddleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))

	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderXRequestID},
		ExposeHeaders:    []string{echo.HeaderXRequestID},
		AllowCredentials: false,
	}))

	e.GET("/healthz", s.Health)

	public := e.Group("/api", middleware.PublicRateLimiter(ctx, middleware.DefaultPublicConfig))

	// Public event discovery
	public.GET("/events/slug/:slug", s.GetEventBySlug)
	public.GET("/events/:id", s.GetEvent)
	public.GET("/events", s.ListPublicEvents)

	// Guest OTP — tighter limiter (each request can enqueue outbound email).
	otp := e.Group("/api", middleware.PublicRateLimiter(ctx, middleware.DefaultOTPConfig))
	otp.POST("/events/:id/register/guest/otp", s.RequestGuestOTP)
	otp.POST("/events/:id/register/guest/otp/verify", s.VerifyGuestOTP)

	// Public guest registration (post-OTP)
	public.POST("/events/:id/register/guest", s.CreateGuestRegistration)
	public.GET("/guest-registrations", s.GetGuestRegistration)
	public.DELETE("/guest-registrations", s.CancelGuestRegistration)

	// Public volunteer info
	public.GET("/events/:id/volunteer/settings", s.GetVolunteerSettings)
	public.GET("/events/:id/volunteer/roles", s.ListVolunteerRoles)

	protected := e.Group("/api", middleware.Auth(), middleware.LoadUser(s.pool, s.queries, s.forgetRegistrationsByEmail), middleware.AuthRateLimiter(ctx, middleware.DefaultAuthConfig))
	protected.GET("/me", s.Me)

	// Host-only event management
	protected.POST("/events", s.CreateEvent)
	protected.GET("/me/events", s.ListMyEvents)
	protected.PATCH("/events/:id", s.UpdateEvent)
	protected.DELETE("/events/:id", s.CancelEvent)

	// User registration routes
	protected.POST("/events/:id/register", s.CreateRegistration)
	protected.GET("/registrations/:id", s.GetRegistration)
	protected.DELETE("/registrations/:id", s.CancelRegistration)
	protected.GET("/me/registrations", s.ListMyRegistrations)

	// Admin/Organizer registration routes
	protected.GET("/events/:id/registrations", s.ListEventRegistrations)
	protected.POST("/events/:id/check-in", s.CheckInByEventToken)
	protected.GET("/registrations/:id/admin", s.GetRegistrationAsAdmin)
	protected.POST("/registrations/:id/approve", s.ApproveRegistration)
	protected.POST("/registrations/:id/reject", s.RejectRegistration)
	protected.PATCH("/registrations/:id/status", s.UpdateRegistrationStatus)
	protected.POST("/registrations/:id/check-in", s.CheckInAttendee)

	// Volunteer application routes (auth user)
	protected.POST("/events/:id/volunteer/apply", s.CreateVolunteerApplication)
	protected.GET("/me/event-staff", s.ListMyEventStaff)
	protected.GET("/me/volunteer-applications", s.ListMyVolunteerApplications)
	protected.GET("/volunteer-applications/:id", s.GetMyVolunteerApplication)
	protected.DELETE("/volunteer-applications/:id", s.CancelVolunteerApplication)

	// Organizer volunteer management
	protected.PUT("/events/:id/volunteer/settings", s.UpsertVolunteerSettings)
	protected.POST("/events/:id/volunteer/roles", s.CreateVolunteerRole)
	protected.DELETE("/events/:id/volunteer/roles/:roleId", s.DeleteVolunteerRole)
	protected.GET("/events/:id/volunteer/applications", s.ListEventVolunteerApplications)
	protected.GET("/events/:id/volunteers", s.ListEventVolunteers)
	protected.GET("/volunteer-applications/:id/admin", s.GetVolunteerApplicationAsAdmin)
	protected.POST("/volunteer-applications/:id/approve", s.ApproveVolunteerApplication)
	protected.POST("/volunteer-applications/:id/reject", s.RejectVolunteerApplication)
}

// Health handles GET /healthz.
// If HEALTHZ_TOKEN is set, requires matching ?pass=. Otherwise public liveness.
func (s *Server) Health(c echo.Context) error {
	if s.healthToken != "" {
		got := c.QueryParam("pass")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.healthToken)) != 1 {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Me handles GET /api/me and returns the authenticated user.
func (s *Server) Me(c echo.Context) error {
	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	return c.JSON(http.StatusOK, dto.NewUserResponse(u))
}
