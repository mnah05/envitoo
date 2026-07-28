package email

import (
	"encoding/json"

	"github.com/google/uuid"
)

// EmailMessage is the JetStream payload published by the outbox drainer.
// It is self-contained so the mailer does not need to hit Postgres.
type EmailMessage struct {
	OutboxID     uuid.UUID       `json:"outbox_id"`
	Recipient    string          `json:"recipient"`
	Template     string          `json:"template"`
	TemplateData json.RawMessage `json:"template_data"`
}

// Template name constants — must match producers in internal/service.
const (
	TemplateOTP                   = "otp"
	TemplateTicketConfirm         = "ticket_confirm"
	TemplateTicketCancel          = "ticket_cancel"
	TemplateTicketStatusChange    = "ticket_status_change"
	TemplateApplicationConfirm    = "application_confirm"
	TemplateApplicationCancel     = "application_cancel"
	TemplateVolunteerStatusChange = "volunteer_status_change"
	TemplateCheckIn               = "check_in"
	TemplateEventCancellation     = "event_cancellation"
)

// NATS subject constants — must match producers in internal/service.
const (
	TopicOTP                   = "emails.otp"
	TopicTicketConfirm         = "emails.ticket.confirmed"
	TopicTicketCancel          = "emails.ticket.cancelled"
	TopicTicketStatusChange    = "emails.ticket.status_changed"
	TopicApplicationConfirm    = "emails.application.confirmed"
	TopicApplicationCancel     = "emails.application.cancelled"
	TopicVolunteerStatusChange = "emails.volunteer.status_changed"
	TopicCheckIn               = "emails.checkin"
	TopicEventCancellation     = "emails.event.cancelled"
	TopicUnknown               = "emails.unknown"
)

// TopicForTemplate maps a template name to its NATS subject.
// Unknown templates fall back to TopicUnknown so publishes never use an empty subject.
var TopicForTemplate = map[string]string{
	TemplateOTP:                   TopicOTP,
	TemplateTicketConfirm:         TopicTicketConfirm,
	TemplateTicketCancel:          TopicTicketCancel,
	TemplateTicketStatusChange:    TopicTicketStatusChange,
	TemplateApplicationConfirm:    TopicApplicationConfirm,
	TemplateApplicationCancel:     TopicApplicationCancel,
	TemplateVolunteerStatusChange: TopicVolunteerStatusChange,
	TemplateCheckIn:               TopicCheckIn,
	TemplateEventCancellation:     TopicEventCancellation,
}

// DefaultTopicFor returns the subject for a template, or TopicUnknown.
func DefaultTopicFor(template string) string {
	if t, ok := TopicForTemplate[template]; ok {
		return t
	}
	return TopicUnknown
}
