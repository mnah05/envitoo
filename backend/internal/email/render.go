package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	texttemplate "text/template"
)

// Renderer turns EmailMessage payloads into subject + HTML/text bodies.
type Renderer struct {
	publicURL string
	html      *template.Template
	text      *texttemplate.Template
}

// NewRenderer parses embedded templates. publicURL is the SPA origin
// (e.g. https://envitoo.app) used for ticket and home links.
func NewRenderer(publicURL string) (*Renderer, error) {
	publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if publicURL == "" {
		return nil, fmt.Errorf("APP_PUBLIC_URL is required")
	}
	if _, err := url.ParseRequestURI(publicURL); err != nil {
		return nil, fmt.Errorf("APP_PUBLIC_URL is invalid: %w", err)
	}

	htmlTmpl, err := template.New("html").ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse html templates: %w", err)
	}
	textTmpl, err := texttemplate.New("text").ParseFS(templateFS, "templates/*.txt")
	if err != nil {
		return nil, fmt.Errorf("parse text templates: %w", err)
	}

	return &Renderer{
		publicURL: publicURL,
		html:      htmlTmpl,
		text:      textTmpl,
	}, nil
}

// Rendered is the output of Renderer.Render.
type Rendered struct {
	Subject string
	HTML    string
	Text    string
}

// Render builds subject and bodies for a known template.
func (r *Renderer) Render(templateName string, data json.RawMessage) (Rendered, error) {
	view, err := buildView(templateName, data, r.publicURL)
	if err != nil {
		return Rendered{}, err
	}

	var htmlBuf, textBuf bytes.Buffer
	if err := r.html.ExecuteTemplate(&htmlBuf, templateName+".html", view); err != nil {
		return Rendered{}, fmt.Errorf("execute html %s: %w", templateName, err)
	}
	if err := r.text.ExecuteTemplate(&textBuf, templateName+".txt", view); err != nil {
		return Rendered{}, fmt.Errorf("execute text %s: %w", templateName, err)
	}

	return Rendered{
		Subject: subjectFor(templateName, view),
		HTML:    htmlBuf.String(),
		Text:    textBuf.String(),
	}, nil
}

// view is the template context shared by HTML and text templates.
type view struct {
	Name        string
	EventTitle  string
	EventID     string
	OTP         string
	NewStatus   string
	RoleName    string
	Title       string
	AccessToken string
	PublicURL   string
	TicketURL   string
	HomeURL     string
	CTAURL      string
	CTALabel    string
}

func buildView(templateName string, raw json.RawMessage, publicURL string) (view, error) {
	v := view{
		PublicURL: publicURL,
		HomeURL:   publicURL + "/home",
	}

	switch templateName {
	case TemplateOTP:
		var d struct {
			OTP        string `json:"otp"`
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode otp data: %w", err)
		}
		v.OTP = d.OTP
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID

	case TemplateTicketConfirm:
		var d struct {
			Name        string `json:"name"`
			EventTitle  string `json:"event_title"`
			EventID     string `json:"event_id"`
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode ticket_confirm data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.AccessToken = d.AccessToken
		if isUsableToken(d.AccessToken) {
			v.TicketURL = publicURL + "/ticket?token=" + url.QueryEscape(d.AccessToken)
			v.CTAURL = v.TicketURL
			v.CTALabel = "View your ticket"
		} else {
			v.CTAURL = v.HomeURL
			v.CTALabel = "View your tickets"
		}

	case TemplateTicketCancel:
		var d struct {
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode ticket_cancel data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.CTAURL = v.HomeURL
		v.CTALabel = "Go to EnviToo"

	case TemplateTicketStatusChange:
		var d struct {
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
			NewStatus  string `json:"new_status"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode ticket_status_change data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.NewStatus = d.NewStatus
		v.CTAURL = v.HomeURL
		v.CTALabel = "View your tickets"

	case TemplateApplicationConfirm:
		var d struct {
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
			RoleName   string `json:"role_name"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode application_confirm data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.RoleName = d.RoleName
		v.CTAURL = v.HomeURL
		v.CTALabel = "Go to EnviToo"

	case TemplateApplicationCancel:
		var d struct {
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode application_cancel data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.CTAURL = v.HomeURL
		v.CTALabel = "Go to EnviToo"

	case TemplateVolunteerStatusChange:
		var d struct {
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
			NewStatus  string `json:"new_status"`
			RoleName   string `json:"role_name"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode volunteer_status_change data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.NewStatus = d.NewStatus
		v.RoleName = d.RoleName
		v.CTAURL = v.HomeURL
		v.CTALabel = "Go to EnviToo"

	case TemplateCheckIn:
		var d struct {
			Name       string `json:"name"`
			EventTitle string `json:"event_title"`
			EventID    string `json:"event_id"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode check_in data: %w", err)
		}
		v.Name = d.Name
		v.EventTitle = d.EventTitle
		v.EventID = d.EventID
		v.CTAURL = v.HomeURL
		v.CTALabel = "Go to EnviToo"

	case TemplateEventCancellation:
		var d struct {
			Title   string `json:"title"`
			EventID string `json:"event_id"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return view{}, fmt.Errorf("decode event_cancellation data: %w", err)
		}
		v.Title = d.Title
		v.EventTitle = d.Title
		v.EventID = d.EventID
		v.CTAURL = publicURL + "/discover"
		v.CTALabel = "Discover events"

	default:
		return view{}, fmt.Errorf("unknown email template %q", templateName)
	}

	return v, nil
}

func subjectFor(templateName string, v view) string {
	switch templateName {
	case TemplateOTP:
		return "Your EnviToo verification code"
	case TemplateTicketConfirm:
		return fmt.Sprintf("You're registered for %s", v.EventTitle)
	case TemplateTicketCancel:
		return fmt.Sprintf("Registration cancelled — %s", v.EventTitle)
	case TemplateTicketStatusChange:
		return fmt.Sprintf("Registration update — %s", v.EventTitle)
	case TemplateApplicationConfirm:
		return fmt.Sprintf("Volunteer application received — %s", v.EventTitle)
	case TemplateApplicationCancel:
		return fmt.Sprintf("Volunteer application cancelled — %s", v.EventTitle)
	case TemplateVolunteerStatusChange:
		return fmt.Sprintf("Volunteer update — %s", v.EventTitle)
	case TemplateCheckIn:
		return fmt.Sprintf("Checked in — %s", v.EventTitle)
	case TemplateEventCancellation:
		return fmt.Sprintf("Event cancelled — %s", v.Title)
	default:
		return "EnviToo notification"
	}
}

func isUsableToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	// Zero UUID means no guest token (signed-in registration).
	return token != "00000000-0000-0000-0000-000000000000"
}
