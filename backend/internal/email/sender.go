package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/resend/resend-go/v3"
)

// Sender delivers rendered emails through Resend.
//
// Note 1: Resend (resend.com) is the email sending provider. It replaces
// older services like SendGrid or SMTP with a modern REST API. The SDK
// handles retries and rate limiting internally.
type Sender struct {
	client *resend.Client
	from   string
}

// NewSender builds a Resend-backed sender. from must be a verified sender address.
func NewSender(apiKey, from string) (*Sender, error) {
	apiKey = strings.TrimSpace(apiKey)
	from = strings.TrimSpace(from)
	if apiKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}
	if from == "" {
		return nil, fmt.Errorf("EMAIL_FROM is required")
	}
	return &Sender{
		client: resend.NewClient(apiKey),
		from:   from,
	}, nil
}

// SendRequest is a fully rendered outbound email.
//
// Note 2: The email is pre-rendered (HTML and Text are supplied) — the sender
// only handles delivery, not templating. This separates concerns: the
// Renderer handles templates, the Sender handles the API call.
type SendRequest struct {
	To             string
	Subject        string
	HTML           string
	Text           string
	IdempotencyKey string
}

// Send delivers one email. IdempotencyKey should be the outbox UUID so NATS
// redeliveries do not create duplicate Resend sends.
//
// Note 3: The IdempotencyKey is passed through to Resend's API. If the same
// key is sent again within a dedup window, Resend returns the previous
// response instead of sending a duplicate email. This is critical for NATS
// at-least-once delivery semantics.
func (s *Sender) Send(ctx context.Context, req SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient is required")
	}
	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if req.HTML == "" && req.Text == "" {
		return fmt.Errorf("html or text body is required")
	}

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{req.To},
		Subject: req.Subject,
		Html:    req.HTML,
		Text:    req.Text,
	}

	opts := &resend.SendEmailOptions{}
	if key := strings.TrimSpace(req.IdempotencyKey); key != "" {
		opts.IdempotencyKey = key
	}

	_, err := s.client.Emails.SendWithOptions(ctx, params, opts)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}
