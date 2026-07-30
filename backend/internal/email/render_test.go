package email

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderer_AllTemplates(t *testing.T) {
	r, err := NewRenderer("https://envitoo.app")
	if err != nil {
		t.Fatal(err)
	}

	// Note 1: Table-driven test covering every email template. Each case
	// specifies the template name, mock data, and expected substrings in both
	// the subject line and the HTML/text body. The body is not matched exactly
	// because Go templates add whitespace and formatting.

	cases := []struct {
		name    string
		data    any
		subjSub string
		bodySub string
	}{
		{
			name: TemplateOTP,
			data: map[string]string{
				"otp": "123456", "name": "Ada", "event_title": "Hack Night", "event_id": "e1",
			},
			subjSub: "verification code",
			bodySub: "123456",
		},
		{
			name: TemplateTicketConfirm,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1",
				"access_token": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			},
			subjSub: "Hack Night",
			bodySub: "/ticket?token=",
		},
		{
			name: TemplateTicketConfirm,
			// Zero UUID = authenticated registration, should link to /home
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1",
				"access_token": "00000000-0000-0000-0000-000000000000",
			},
			subjSub: "Hack Night",
			bodySub: "/home",
		},
		{
			name: TemplateTicketCancel,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1",
			},
			subjSub: "cancelled",
			bodySub: "Hack Night",
		},
		{
			name: TemplateTicketStatusChange,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1", "new_status": "approved",
			},
			subjSub: "Registration update",
			bodySub: "approved",
		},
		{
			name: TemplateApplicationConfirm,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1", "role_name": "Door",
			},
			subjSub: "Volunteer application received",
			bodySub: "Door",
		},
		{
			name: TemplateApplicationCancel,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1",
			},
			subjSub: "Volunteer application cancelled",
			bodySub: "Hack Night",
		},
		{
			name: TemplateVolunteerStatusChange,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1",
				"new_status": "approved", "role_name": "Door",
			},
			subjSub: "Volunteer update",
			bodySub: "approved",
		},
		{
			name: TemplateCheckIn,
			data: map[string]string{
				"name": "Ada", "event_title": "Hack Night", "event_id": "e1",
			},
			subjSub: "Checked in",
			bodySub: "checked in",
		},
		{
			name: TemplateEventCancellation,
			data: map[string]string{
				"title": "Hack Night", "event_id": "e1",
			},
			subjSub: "Event cancelled",
			bodySub: "/discover",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/"+tc.subjSub, func(t *testing.T) {
			raw, err := json.Marshal(tc.data)
			if err != nil {
				t.Fatal(err)
			}
			out, err := r.Render(tc.name, raw)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(strings.ToLower(out.Subject), strings.ToLower(tc.subjSub)) {
				t.Fatalf("subject %q missing %q", out.Subject, tc.subjSub)
			}
			if !strings.Contains(out.HTML, tc.bodySub) && !strings.Contains(strings.ToLower(out.HTML), strings.ToLower(tc.bodySub)) {
				t.Fatalf("html missing %q", tc.bodySub)
			}
			if !strings.Contains(out.Text, tc.bodySub) && !strings.Contains(strings.ToLower(out.Text), strings.ToLower(tc.bodySub)) {
				t.Fatalf("text missing %q", tc.bodySub)
			}
			if out.HTML == "" || out.Text == "" {
				t.Fatal("empty body")
			}
		})
	}
}

func TestRenderer_UnknownTemplate(t *testing.T) {
	r, err := NewRenderer("https://envitoo.app")
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Render("nope", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultTopicFor(t *testing.T) {
	if DefaultTopicFor(TemplateOTP) != TopicOTP {
		t.Fatal(DefaultTopicFor(TemplateOTP))
	}
	if DefaultTopicFor("missing") != TopicUnknown {
		t.Fatal(DefaultTopicFor("missing"))
	}
}
