package email

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// sendTimeout bounds a single Resend call so a hung request cannot block
// the JetStream ack deadline forever.
const sendTimeout = 45 * time.Second

// Consumer renders and sends one email message from the JetStream consumer.
// It binds the Renderer (templates) to the Sender (Resend delivery) and
// decides per message whether to ack, nak, or term based on whether the
// failure is permanent or transient.
type Consumer struct {
	renderer *Renderer
	sender   *Sender
}

// NewConsumer builds the mailer-side message handler.
func NewConsumer(renderer *Renderer, sender *Sender) *Consumer {
	return &Consumer{renderer: renderer, sender: sender}
}

// Handle processes a single email message from the JetStream consumer.
// Each send is bound with sendTimeout so a hung Resend call cannot block
// the ack deadline forever.
func (c *Consumer) Handle(ctx context.Context, msg jetstream.Msg, log *slog.Logger) {
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	var payload EmailMessage
	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		log.Error("mailer_invalid_payload", "subject", msg.Subject(), "error", err)
		// Poison message — acknowledge so it does not retry forever.
		_ = msg.Ack()
		return
	}

	msgLog := log.With(
		"outbox_id", payload.OutboxID,
		"template", payload.Template,
		"topic", msg.Subject(),
	)

	rendered, err := c.renderer.Render(payload.Template, payload.TemplateData)
	if err != nil {
		msgLog.Error("mailer_render_failed", "error", err)
		if isPermanentRenderErr(err) {
			_ = msg.Ack()
			return
		}
		_ = msg.Nak()
		return
	}

	err = c.sender.Send(sendCtx, SendRequest{
		To:             payload.Recipient,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: payload.OutboxID.String(),
	})
	if err != nil {
		msgLog.Error("mailer_send_failed", "error", err)
		if isPermanentSendErr(err) {
			_ = msg.Term()
			return
		}
		_ = msg.Nak()
		return
	}

	if err := msg.Ack(); err != nil {
		msgLog.Error("mailer_ack_failed", "error", err)
		return
	}
	msgLog.Info("mailer_send_ok")
}

// isPermanentRenderErr returns true for template errors that will never
// succeed on retry (e.g. unknown template name, decode failure). The
// message is Ack'd (not Nak'd) so it is removed from the stream.
func isPermanentRenderErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown email template") ||
		strings.Contains(msg, "decode ")
}

// isPermanentSendErr returns true for Resend API errors that indicate a
// bad request (4xx) rather than a transient failure (5xx/timeout).
// Permanent errors call msg.Term() to move them to the terminal backlog;
// transient errors call msg.Nak() for redelivery.
func isPermanentSendErr(err error) bool {
	if err == nil {
		return false
	}
	// Resend returns API errors with status embedded in the message.
	// Treat clear client errors as terminal; everything else is retryable.
	msg := strings.ToLower(err.Error())
	if errors.Is(err, context.Canceled) {
		return false
	}
	permanentHints := []string{
		"status code: 400",
		"status code: 401",
		"status code: 403",
		"status code: 404",
		"status code: 422",
		"invalid api key",
		"validation_error",
	}
	for _, h := range permanentHints {
		if strings.Contains(msg, h) {
			return true
		}
	}
	return false
}
