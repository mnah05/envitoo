package service

import "errors"

var (
	ErrEventNotFound                = errors.New("event not found")
	ErrRegistrationNotFound         = errors.New("registration not found")
	ErrRegistrationInvalidStatus    = errors.New("registration status transition not allowed")
	ErrInvalidEvent                 = errors.New("invalid event")
	ErrConflict                     = errors.New("registration already exists")
	ErrEventFull                    = errors.New("event is full")
	ErrEventNotOpen                 = errors.New("event not open for registration")
	ErrVolunteerSettingsNotFound    = errors.New("volunteer settings not found")
	ErrVolunteerNotOpen             = errors.New("volunteer applications are not open")
	ErrVolunteerMaxReached          = errors.New("volunteer application cap reached")
	ErrVolunteerDuplicate           = errors.New("already applied for this event")
	ErrVolunteerRoleNotFound        = errors.New("volunteer role not found")
	ErrVolunteerAppNotFound         = errors.New("volunteer application not found")
	ErrVolunteerInvalidStatus       = errors.New("volunteer application status transition not allowed")
	ErrVolunteerRoleDuplicate       = errors.New("volunteer role name already exists")
	ErrVolunteerRoleInUse           = errors.New("volunteer role has applications")
	ErrEventStaffForbidden          = errors.New("event staff permission denied")
	ErrVolunteerAdminRoleRestricted = errors.New("only the organizer can moderate admin-role applications")

	// Guest email OTP
	ErrOTPInvalid            = errors.New("invalid or expired otp")
	ErrOTPRateLimited        = errors.New("otp resend rate limited")
	ErrDependencyUnavailable = errors.New("required dependency unavailable")
	ErrGuestEmailNotVerified = errors.New("guest email not verified")
)
