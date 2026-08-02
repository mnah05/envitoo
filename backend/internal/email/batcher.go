package email

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/resend/resend-go/v3"
	"golang.org/x/time/rate"
)

// maxBatch is Resend's batch endpoint limit.
const maxBatch = 100

// fetchWindow caps how long we wait for a full batch before sending
// whatever messages have accumulated during low-traffic periods.
const fetchWindow = 500 * time.Millisecond

// resendRate is the rate at which we call Resend's batch endpoint.
// 4 req/sec leaves a 1 req/sec margin under the documented 5 req/sec limit.
const resendRate = 4

// nakDelay is the delay passed to NakWithDelay after a transient failure
// so Resend's rate limit has time to recover before redelivery.
const nakDelay = 2 * time.Second

// sendTimeout bounds a single batch Resend call.
const sendTimeout = 45 * time.Second

// Batcher consumes messages from a NATS JetStream pull consumer, renders
// them, and sends them to Resend in batches of up to 100 using a single
// API call per batch. The batch endpoint reduces request COUNT against
// Resend's hard 5 req/sec team limit — converting ~85 individual calls
// per second at peak into at most 4 batch calls.
//
// Concurrency model: single goroutine. No worker pools. The rate limiter
// gates every batch call.
type Batcher struct {
	renderer *Renderer
	sender   BatchSender
	log      *slog.Logger
}

// NewBatcher creates a pull-based batch mailer.
func NewBatcher(renderer *Renderer, sender BatchSender, log *slog.Logger) *Batcher {
	return &Batcher{renderer: renderer, sender: sender, log: log}
}

// Run runs the pull-consume → render → batch-send loop until ctx is
// cancelled. It returns ctx.Err() on graceful shutdown or the first
// non-retryable error.
func (b *Batcher) Run(ctx context.Context, consumer jetstream.Consumer) error {
	limiter := rate.NewLimiter(resendRate, 1)

	for {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}

		msgs, err := consumer.Fetch(maxBatch, jetstream.FetchMaxWait(fetchWindow))
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			b.log.WarnContext(ctx, "fetch_failed", "error", err)
			continue
		}

		// Drain the MessageBatch channel into a slice.
		var batch []jetstream.Msg
		for msg := range msgs.Messages() {
			batch = append(batch, msg)
		}
		if err := msgs.Error(); err != nil {
			b.log.WarnContext(ctx, "fetch_batch_error", "error", err)
		}
		if len(batch) == 0 {
			continue
		}

		b.processBatch(ctx, batch)
	}
}

// processBatch renders every message, sends the batch through Resend, and
// Acks/Naks/Terms based on the response.
func (b *Batcher) processBatch(ctx context.Context, msgs []jetstream.Msg) {
	sendReqs, validMsgs := b.buildSendRequests(ctx, msgs)
	if len(sendReqs) == 0 {
		return
	}

	b.log.Info("batch_send_start", "n", len(sendReqs))

	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	resp, err := b.sender.SendBatch(sendCtx, sendReqs)
	if err != nil {
		b.log.ErrorContext(ctx, "batch_send_failed", "n", len(sendReqs), "error", err)
		for _, msg := range validMsgs {
			_ = msg.NakWithDelay(nakDelay)
		}
		return
	}

	if len(resp.Errors) > 0 {
		b.handlePartialFailure(ctx, validMsgs, resp.Errors)
	} else {
		for _, msg := range validMsgs {
			_ = msg.Ack()
		}
	}

	b.log.Info("batch_send_done", "sent", len(sendReqs), "errors", len(resp.Errors))
}

// buildSendRequests unmarshals and renders each message. Messages that fail
// to render are Term'd (poison) or Nak'd (transient) immediately. Valid
// renders are collected into SendRequests for the batch call.
func (b *Batcher) buildSendRequests(ctx context.Context, msgs []jetstream.Msg) ([]SendRequest, []jetstream.Msg) {
	sendReqs := make([]SendRequest, 0, len(msgs))
	validMsgs := make([]jetstream.Msg, 0, len(msgs))

	for _, msg := range msgs {
		var payload EmailMessage
		if err := json.Unmarshal(msg.Data(), &payload); err != nil {
			b.log.ErrorContext(ctx, "invalid_payload", "subject", msg.Subject(), "error", err)
			_ = msg.Term()
			continue
		}

		rendered, err := b.renderer.Render(payload.Template, payload.TemplateData)
		if err != nil {
			if isPermanentRenderErr(err) {
				b.log.ErrorContext(ctx, "render_failed_permanent", "template", payload.Template, "error", err)
				_ = msg.Term()
			} else {
				b.log.WarnContext(ctx, "render_failed_transient", "template", payload.Template, "error", err)
				_ = msg.NakWithDelay(nakDelay)
			}
			continue
		}

		sendReqs = append(sendReqs, SendRequest{
			To:             payload.Recipient,
			Subject:        rendered.Subject,
			HTML:           rendered.HTML,
			Text:           rendered.Text,
			IdempotencyKey: payload.OutboxID.String(),
		})
		validMsgs = append(validMsgs, msg)
	}

	return sendReqs, validMsgs
}

// handlePartialFailure Acks successful messages and Naks failed ones by
// their batch index.
func (b *Batcher) handlePartialFailure(ctx context.Context, msgs []jetstream.Msg, batchErrs []resend.BatchError) {
	failIdx := make(map[int]bool, len(batchErrs))
	for _, be := range batchErrs {
		failIdx[be.Index] = true
		b.log.WarnContext(ctx, "batch_email_failed",
			"index", be.Index,
			"message", be.Message,
		)
	}

	for i, msg := range msgs {
		if failIdx[i] {
			_ = msg.NakWithDelay(nakDelay)
		} else {
			_ = msg.Ack()
		}
	}
}

// isPermanentRenderErr returns true for template errors that will never
// succeed on retry (e.g. unknown template name, decode failure). The
// message is Term'd so it is removed from the stream.
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
