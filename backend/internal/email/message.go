package email

import (
	"encoding/json"

	"github.com/google/uuid"
)

// EmailMessage is the JetStream payload published by the outbox drainer.
// It is self-contained so the mailer does not need to hit Postgres.
//
// Note 1: The outbox drainer reads from the email_outbox table in Postgres
// and publishes EmailMessage to NATS JetStream. The mailer process then
// consumes these messages and sends them via Resend. This decoupling means
// the API server never blocks on email delivery.
//
// Note 2: TemplateData uses json.RawMessage to defer JSON decoding. The
// mailer interprets this data based on the template name, so different
// templates can have different data shapes without changing the message
// envelope.
type EmailMessage struct {
	OutboxID     uuid.UUID       `json:"outbox_id"`
	Recipient    string          `json:"recipient"`
	Template     string          `json:"template"`
	TemplateData json.RawMessage `json:"template_data"`
}

// Template name constants — must match producers in internal/service.
//
// Note 3: These constants are the glue between the service layer (which
// enqueues outbox rows with a template name) and the mailer (which renders
// and sends). Both sides import the same package to stay in sync.
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
//
// Note 4: NATS subjects use dot-separated segments (emails.ticket.confirmed).
// This is NATS' native topic hierarchy, which allows consumers to subscribe
// with wildcards like "emails.>" (all email topics) or "emails.ticket.*"
// (all ticket-related emails).
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
//
// Note 5: A map is used instead of a switch so callers can also iterate or
// check membership at runtime, which is useful for testing and observability.
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
