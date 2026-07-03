package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/dto"
)

// EventStaffRole describes how a user may act on an event.
type EventStaffRole int

const (
	EventStaffNone EventStaffRole = iota
	EventStaffOrganizer
	EventStaffVolunteerAdmin
)

// EventAccessService resolves organizer and volunteer-admin permissions.
type EventAccessService struct {
	queries *db.Queries
}

// NewEventAccessService creates an EventAccessService.
func NewEventAccessService(queries *db.Queries) *EventAccessService {
	return &EventAccessService{queries: queries}
}

// ResolveStaffRole returns the caller's staff role for an event.
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

// RequireEventStaff returns ErrEventNotFound when the user has no staff access.
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
// only when the event requires approval.
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
// fields volunteer admins are not allowed to change.
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
