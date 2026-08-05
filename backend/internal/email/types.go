package email

// EmailData is the payload contract between the service layer (which
// enqueues outbox rows) and the mailer (which renders and sends them).
//
// Each concrete type carries its own template name and NATS topic, so a
// producer can never enqueue an email under a mismatched template/topic
// pair. The mailer decodes the JSON payload back into these same types
// (see render.go) — a single definition on both sides of the pipeline.
type EmailData interface {
	Template() string
	Topic() string
}

// OTPData is the payload for the guest verification code email.
type OTPData struct {
	OTP        string `json:"otp"`
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
}

func (OTPData) Template() string { return TemplateOTP }
func (OTPData) Topic() string    { return TopicOTP }

// TicketConfirmData is the payload for a confirmed ticket.
type TicketConfirmData struct {
	Name        string `json:"name"`
	EventTitle  string `json:"event_title"`
	EventID     string `json:"event_id"`
	AccessToken string `json:"access_token"`
}

func (TicketConfirmData) Template() string { return TemplateTicketConfirm }
func (TicketConfirmData) Topic() string    { return TopicTicketConfirm }

// TicketCancelData is the payload for a cancelled registration.
type TicketCancelData struct {
	Name        string `json:"name"`
	EventTitle  string `json:"event_title"`
	EventID     string `json:"event_id"`
	AccessToken string `json:"access_token"`
}

func (TicketCancelData) Template() string { return TemplateTicketCancel }
func (TicketCancelData) Topic() string    { return TopicTicketCancel }

// TicketStatusChangeData is the payload for a registration status update.
type TicketStatusChangeData struct {
	Name        string `json:"name"`
	EventTitle  string `json:"event_title"`
	EventID     string `json:"event_id"`
	NewStatus   string `json:"new_status"`
	AccessToken string `json:"access_token"`
}

func (TicketStatusChangeData) Template() string { return TemplateTicketStatusChange }
func (TicketStatusChangeData) Topic() string    { return TopicTicketStatusChange }

// ApplicationConfirmData is the payload for a received volunteer application.
type ApplicationConfirmData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
	RoleName   string `json:"role_name"`
}

func (ApplicationConfirmData) Template() string { return TemplateApplicationConfirm }
func (ApplicationConfirmData) Topic() string    { return TopicApplicationConfirm }

// ApplicationCancelData is the payload for a cancelled volunteer application.
type ApplicationCancelData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
}

func (ApplicationCancelData) Template() string { return TemplateApplicationCancel }
func (ApplicationCancelData) Topic() string    { return TopicApplicationCancel }

// VolunteerStatusChangeData is the payload for a volunteer status update.
type VolunteerStatusChangeData struct {
	Name       string `json:"name"`
	EventTitle string `json:"event_title"`
	EventID    string `json:"event_id"`
	NewStatus  string `json:"new_status"`
	RoleName   string `json:"role_name"`
}

func (VolunteerStatusChangeData) Template() string { return TemplateVolunteerStatusChange }
func (VolunteerStatusChangeData) Topic() string    { return TopicVolunteerStatusChange }

// CheckInData is the payload for the attendance check-in confirmation email.
type CheckInData struct {
	Name        string `json:"name"`
	EventTitle  string `json:"event_title"`
	EventID     string `json:"event_id"`
	AccessToken string `json:"access_token"`
}

func (CheckInData) Template() string { return TemplateCheckIn }
func (CheckInData) Topic() string    { return TopicCheckIn }

// EventCancellationData is the payload for the event cancellation notice.
type EventCancellationData struct {
	Title   string `json:"title"`
	EventID string `json:"event_id"`
}

func (EventCancellationData) Template() string { return TemplateEventCancellation }
func (EventCancellationData) Topic() string    { return TopicEventCancellation }
