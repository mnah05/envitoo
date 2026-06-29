package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/mnah05/pulse/internal/db"
	"github.com/mnah05/pulse/internal/dto"
	"github.com/mnah05/pulse/internal/middleware"
	"github.com/mnah05/pulse/internal/service"
)

// Server holds the HTTP handlers and their dependencies.
type Server struct {
	pool          *pgxpool.Pool
	queries       *db.Queries
	log           *slog.Logger
	validate      *validator.Validate
	events        *service.EventService
	registrations *service.RegistrationService
	volunteers    *service.VolunteerService
}

// New creates a Server with all services initialized.
func New(pool *pgxpool.Pool, queries *db.Queries, log *slog.Logger) *Server {
	validate := dto.NewValidator()
	access := service.NewEventAccessService(queries)
	return &Server{
		pool:          pool,
		queries:       queries,
		log:           log,
		validate:      validate,
		events:        service.NewEventService(pool, queries, access, log),
		registrations: service.NewRegistrationService(pool, queries, access, log),
		volunteers:    service.NewVolunteerService(pool, queries, access, log),
	}
}

// Register wires all routes onto the given Echo instance.
func (s *Server) Register(e *echo.Echo) {
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.RequestID())

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log := s.log
			if rid := c.Response().Header().Get(echo.HeaderXRequestID); rid != "" {
				log = log.With("request_id", rid)
			}
			c.SetRequest(c.Request().WithContext(middleware.WithLogger(c.Request().Context(), log)))
			return next(c)
		}
	})

	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderXRequestID},
		ExposeHeaders:    []string{echo.HeaderXRequestID},
		AllowCredentials: false,
	}))

	e.Use(echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStoreWithConfig(
		echoMiddleware.RateLimiterMemoryStoreConfig{
			Rate:      30,
			Burst:     60,
			ExpiresIn: 3 * time.Minute,
		},
	)))

	e.GET("/healthz", s.Health)
	protected := e.Group("/api", middleware.Auth(), middleware.LoadUser(s.pool, s.queries))
	protected.GET("/me", s.Me)

	// Public event discovery
	e.GET("/api/events/slug/:slug", s.GetEventBySlug)
	e.GET("/api/events/:id", s.GetEvent)
	e.GET("/api/events", s.ListPublicEvents)

	// Public guest registration
	e.POST("/api/events/:id/register/guest", s.CreateGuestRegistration)
	e.GET("/api/guest-registrations", s.GetGuestRegistration)
	e.DELETE("/api/guest-registrations", s.CancelGuestRegistration)

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
	protected.GET("/registrations/:id/admin", s.GetRegistrationAsAdmin)
	protected.POST("/registrations/:id/approve", s.ApproveRegistration)
	protected.POST("/registrations/:id/reject", s.RejectRegistration)
	protected.PATCH("/registrations/:id/status", s.UpdateRegistrationStatus)
	protected.POST("/registrations/:id/check-in", s.CheckInAttendee)

	// Public volunteer info
	e.GET("/api/events/:id/volunteer/settings", s.GetVolunteerSettings)
	e.GET("/api/events/:id/volunteer/roles", s.ListVolunteerRoles)

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
func (s *Server) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Me handles GET /api/me and returns the authenticated user.
func (s *Server) Me(c echo.Context) error {
	u, ok := middleware.UserFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	return c.JSON(http.StatusOK, u)
}
