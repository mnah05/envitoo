package service

// Template names used for email outbox. Each template corresponds to an
// HTML template file on the mailer worker side. The template name stored
// in the outbox row tells the mailer which template to render.
const (
	EmailTemplateOTP                   = "otp"
	EmailTemplateTicketConfirm         = "ticket_confirm"
	EmailTemplateTicketCancel          = "ticket_cancel"
	EmailTemplateTicketStatusChange    = "ticket_status_change"
	EmailTemplateApplicationConfirm    = "application_confirm"
	EmailTemplateApplicationCancel     = "application_cancel"
	EmailTemplateVolunteerStatusChange = "volunteer_status_change"
	EmailTemplateCheckIn               = "check_in"
	EmailTemplateEventCancellation     = "event_cancellation"
)

// NATS topics for the async email pipeline. The backend writes an outbox
// row + publishes to NATS; the mailer worker consumes the NATS message
// and sends via Resend. Separating topic from template allows consumers
// to subscribe to a subset of email types.
const (
	EmailTopicOTP                   = "emails.otp"
	EmailTopicTicketConfirm         = "emails.ticket.confirmed"
	EmailTopicTicketCancel          = "emails.ticket.cancelled"
	EmailTopicTicketStatusChange    = "emails.ticket.status_changed"
	EmailTopicApplicationConfirm    = "emails.application.confirmed"
	EmailTopicApplicationCancel     = "emails.application.cancelled"
	EmailTopicVolunteerStatusChange = "emails.volunteer.status_changed"
	EmailTopicCheckIn               = "emails.checkin"
	EmailTopicEventCancellation     = "emails.event.cancelled"
)

// Data structs serialised to JSON and stored in the outbox template_data
// column. The mailer worker unmarshals these and passes them to the
// appropriate HTML template for rendering.
type ticketConfirmData struct {
	Name        string `json:"name"`
	EventTitle  string `json:"event_title"`
	EventID     string `json:"event_id"`
	AccessToken string `json:"access_token"`
}

type ticketCancelData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
}

type ticketStatusChangeData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
	NewStatus  string `json:"new_status"`
}

type applicationConfirmData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
	RoleName   string `json:"role_name"`
}

type applicationCancelData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
}

type volunteerStatusChangeData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
	NewStatus  string `json:"new_status"`
	RoleName   string `json:"role_name"`
}

type checkInData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
}

type eventCancellationData struct {
	Title   string `json:"title"`
	EventID string `json:"event_id"`
}
