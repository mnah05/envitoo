package service

// Template names
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

// NATS topics
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
