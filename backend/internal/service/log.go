package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/mnah05/envitoo/internal/middleware"
)

// Service-level loggers attach a "component" attribute so log entries from
// different services are distinguishable in the aggregated log stream.
func (s *RegistrationService) logger(ctx context.Context) *slog.Logger {
	return middleware.LoggerFromContext(ctx).With("component", "registration")
}

func (s *EventService) logger(ctx context.Context) *slog.Logger {
	return middleware.LoggerFromContext(ctx).With("component", "event")
}

func (s *VolunteerService) logger(ctx context.Context) *slog.Logger {
	return middleware.LoggerFromContext(ctx).With("component", "volunteer")
}

// logOpOutcome records success or a classified failure for a mutating op.
// Domain errors (expected business-rule violations) are logged at Warn level;
// unexpected errors (DB down, etc.) are logged at Error level.
// Call with err == nil for success, err != nil for failure.
func logOpOutcome(log *slog.Logger, op string, err error, attrs ...any) {
	if err == nil {
		log.Info(op+"_ok", attrs...)
		return
	}
	if reason, ok := domainFailReason(err); ok {
		a := append(attrs, "reason", reason, "error", err)
		log.Warn(op+"_rejected", a...)
		return
	}
	a := append(attrs, "error", err)
	log.Error(op+"_failed", a...)
}

// domainFailReason classifies a sentinel error into a short string for
// structured logging. The second return value is false for unrecognised
// errors (which should be logged at Error level instead of Warn).
func domainFailReason(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrEventNotFound):
		return "event_not_found", true
	case errors.Is(err, ErrRegistrationNotFound):
		return "registration_not_found", true
	case errors.Is(err, ErrRegistrationInvalidStatus):
		return "invalid_status", true
	case errors.Is(err, ErrInvalidEvent):
		return "invalid_event", true
	case errors.Is(err, ErrConflict):
		return "conflict", true
	case errors.Is(err, ErrEventFull):
		return "event_full", true
	case errors.Is(err, ErrEventNotOpen):
		return "event_not_open", true
	case errors.Is(err, ErrVolunteerSettingsNotFound):
		return "volunteer_settings_not_found", true
	case errors.Is(err, ErrVolunteerNotOpen):
		return "volunteer_not_open", true
	case errors.Is(err, ErrVolunteerMaxReached):
		return "volunteer_max_reached", true
	case errors.Is(err, ErrVolunteerDuplicate):
		return "volunteer_duplicate", true
	case errors.Is(err, ErrVolunteerRoleNotFound):
		return "volunteer_role_not_found", true
	case errors.Is(err, ErrVolunteerAppNotFound):
		return "volunteer_app_not_found", true
	case errors.Is(err, ErrVolunteerInvalidStatus):
		return "volunteer_invalid_status", true
	case errors.Is(err, ErrVolunteerRoleDuplicate):
		return "volunteer_role_duplicate", true
	case errors.Is(err, ErrVolunteerRoleInUse):
		return "volunteer_role_in_use", true
	case errors.Is(err, ErrEventStaffForbidden):
		return "staff_forbidden", true
	case errors.Is(err, ErrVolunteerAdminRoleRestricted):
		return "admin_role_restricted", true
	case errors.Is(err, ErrOTPInvalid):
		return "otp_invalid", true
	case errors.Is(err, ErrOTPRateLimited):
		return "otp_rate_limited", true
	case errors.Is(err, ErrDependencyUnavailable):
		return "dependency_unavailable", true
	case errors.Is(err, ErrGuestEmailNotVerified):
		return "guest_email_not_verified", true
	default:
		return "", false
	}
}
