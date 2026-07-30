package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
)

// EventStaffRole describes how a user may act on an event. iota generates
// successive integer values: None=0, Organizer=1, VolunteerAdmin=2. The
// zero value (None) is the default, so an unset role safely means "no access".
type EventStaffRole int

const (
	EventStaffNone EventStaffRole = iota
	EventStaffOrganizer
	EventStaffVolunteerAdmin
)

// EventAccessService resolves organizer and volunteer-admin permissions.
// It is a stateless query object — injected into the three service types
// so they share one permission check implementation.
type EventAccessService struct {
	queries *db.Queries
}

// NewEventAccessService creates an EventAccessService.
func NewEventAccessService(queries *db.Queries) *EventAccessService {
	return &EventAccessService{queries: queries}
}

// ResolveStaffRole returns the caller's staff role for an event. A single
// DB round-trip checks organizer ownership first (most common case is the
// organizer), then falls through to the volunteer-admin check.
func (s *EventAccessService) ResolveStaffRole(ctx context.Context, eventID, userID uuid.UUID) (EventStaffRole, error) {
	isOrganizer, err := s.queries.IsEventOrganizer(ctx, db.IsEventOrganizerParams{
		ID:          eventID,
		OrganizerID: userID,
	})
	if err != nil {
		return EventStaffNone, fmt.Errorf("check event organizer: %w", err)
	}
	if isOrganizer {
		return EventStaffOrganizer, nil
	}

	isAdmin, err := s.queries.IsVolunteerEventAdmin(ctx, db.IsVolunteerEventAdminParams{
		EventID: eventID,
		UserID:  userID,
	})
	if err != nil {
		return EventStaffNone, fmt.Errorf("check volunteer event admin: %w", err)
	}
	if isAdmin {
		return EventStaffVolunteerAdmin, nil
	}
	return EventStaffNone, nil
}

// RequireEventStaff returns ErrEventNotFound when the user has no staff
// access. Note the deliberate use of ErrEventNotFound (not a custom
// "forbidden" error) — this leaks no information about whether the event
// exists to unprivileged callers.
func (s *EventAccessService) RequireEventStaff(ctx context.Context, eventID, userID uuid.UUID) (EventStaffRole, error) {
	role, err := s.ResolveStaffRole(ctx, eventID, userID)
	if err != nil {
		return EventStaffNone, err
	}
	if role == EventStaffNone {
		return EventStaffNone, fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
	}
	return role, nil
}

// RequireRegistrationModeration allows organizers always and volunteer admins
// only when the event requires approval. Events that do not require approval
// skip the moderation queue entirely — there are no pending registrations to
// moderate, so volunteer admins get ErrEventStaffForbidden.
func (s *EventAccessService) RequireRegistrationModeration(ctx context.Context, eventID, userID uuid.UUID) error {
	role, err := s.RequireEventStaff(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if role == EventStaffOrganizer {
		return nil
	}

	event, err := s.queries.GetEventByID(ctx, eventID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("event %s: %w", eventID, ErrEventNotFound)
		}
		return fmt.Errorf("get event: %w", err)
	}
	if !event.RequiresApproval {
		return fmt.Errorf("event %s: %w", eventID, ErrEventStaffForbidden)
	}
	return nil
}

// HasDisallowedStaffUpdateFields returns true when a PATCH body includes
// fields volunteer admins are not allowed to change. Volunteer admins may
// update description, cover image, venue details, and meeting URL, but
// not structural fields like capacity, pricing, visibility, or category.
func HasDisallowedStaffUpdateFields(req dto.UpdateEventRequest) bool {
	return req.StartsAt != nil ||
		req.EndsAt != nil ||
		req.Capacity != nil ||
		req.PriceInPaise != nil ||
		req.Currency != nil ||
		req.RequiresApproval != nil ||
		req.IsVirtual != nil ||
		req.Category != nil ||
		req.Visibility != nil
}
