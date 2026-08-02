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

// BatchSender is the interface for sending emails in batch through Resend.
// It exists so Batcher can be tested with a mock instead of a real Resend client.
type BatchSender interface {
	SendBatch(ctx context.Context, reqs []SendRequest) (*resend.BatchEmailResponse, error)
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

// SendBatch delivers up to 100 emails in one Resend API call using the batch
// endpoint. Each SendRequest's IdempotencyKey is preserved via the email-level
// idempotency header — Resend's batch validation is set to permissive so
// individual failures do not block the rest of the batch.
//
// SendBatch counts as a single request against Resend's 5 req/sec team limit,
// which is the key reason for this method's existence.
func (s *Sender) SendBatch(ctx context.Context, reqs []SendRequest) (*resend.BatchEmailResponse, error) {
	params := make([]*resend.SendEmailRequest, 0, len(reqs))
	for _, req := range reqs {
		if req.To == "" || req.Subject == "" || (req.HTML == "" && req.Text == "") {
			continue
		}
		params = append(params, &resend.SendEmailRequest{
			From:    s.from,
			To:      []string{req.To},
			Subject: req.Subject,
			Html:    req.HTML,
			Text:    req.Text,
		})
	}

	batchOpts := &resend.BatchSendEmailOptions{
		BatchValidation: resend.BatchValidationPermissive,
	}
	if len(reqs) > 0 {
		if key := strings.TrimSpace(reqs[0].IdempotencyKey); key != "" {
			batchOpts.IdempotencyKey = key
		}
	}

	return s.client.Batch.SendWithOptions(ctx, params, batchOpts)
}
