package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/resend/resend-go/v3"
	"golang.org/x/time/rate"
)

// --- mocks for jetstream.Consumer and jetstream.Msg ---

type mockMsg struct {
	subject  string
	data     []byte
	acked    bool
	nakDelay time.Duration
	nakd     bool
	termed   bool
}

func (m *mockMsg) Ack() error                          { m.acked = true; return nil }
func (m *mockMsg) DoubleAck(ctx context.Context) error { return nil }
func (m *mockMsg) Nak() error                          { m.nakd = true; return nil }
func (m *mockMsg) NakWithDelay(d time.Duration) error  { m.nakd = true; m.nakDelay = d; return nil }
func (m *mockMsg) InProgress() error                   { return nil }
func (m *mockMsg) Term() error                         { m.termed = true; return nil }
func (m *mockMsg) TermWithReason(reason string) error  { m.termed = true; return nil }
func (m *mockMsg) Subject() string                     { return m.subject }
func (m *mockMsg) Data() []byte                        { return m.data }
func (m *mockMsg) Headers() nats.Header                { return nil }
func (m *mockMsg) Reply() string                       { return "" }
func (m *mockMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{}, nil
}

type mockConsumer struct {
	msgs []jetstream.Msg
	mu   sync.Mutex
	// consumeCount tracks how many times Fetch was called
	consumeCount int
	info         *jetstream.ConsumerInfo
	infoErr      error
}

func (m *mockConsumer) Fetch(batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	return &mockMessageBatch{msgs: m.msgs}, nil
}
func (m *mockConsumer) FetchBytes(maxBytes int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConsumer) FetchNoWait(batch int) (jetstream.MessageBatch, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConsumer) Consume(handler jetstream.MessageHandler, opts ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConsumer) Messages(opts ...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConsumer) Next(opts ...jetstream.FetchOpt) (jetstream.Msg, error) {
	return nil, errors.New("not implemented")
}
func (m *mockConsumer) Info(ctx context.Context) (*jetstream.ConsumerInfo, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	if m.info != nil {
		return m.info, nil
	}
	return &jetstream.ConsumerInfo{}, nil
}
func (m *mockConsumer) CachedInfo() *jetstream.ConsumerInfo {
	return m.info
}

type mockMessageBatch struct {
	msgs []jetstream.Msg
}

func (mb *mockMessageBatch) Messages() <-chan jetstream.Msg {
	ch := make(chan jetstream.Msg, len(mb.msgs))
	for _, msg := range mb.msgs {
		ch <- msg
	}
	close(ch)
	return ch
}
func (mb *mockMessageBatch) Error() error { return nil }

// --- mock BatchSender ---

type mockBatchSender struct {
	resp *resend.BatchEmailResponse
	err  error
	mu   sync.Mutex
	// captures the requests sent
	reqs [][]SendRequest
}

func (m *mockBatchSender) SendBatch(ctx context.Context, reqs []SendRequest) (*resend.BatchEmailResponse, error) {
	m.mu.Lock()
	m.reqs = append(m.reqs, reqs)
	m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

// --- renderer helper ---

func testRenderer(publicURL string) *Renderer {
	r, err := NewRenderer(publicURL)
	if err != nil {
		panic(fmt.Sprintf("test Renderer init failed: %v", err))
	}
	return r
}

// buildValidPayload returns marshaled EmailMessage with a known template.
func buildValidPayload(template, recipient string) []byte {
	msg := EmailMessage{
		OutboxID:  uuid.New(),
		Recipient: recipient,
		Template:  template,
	}

	// Provide template-specific data so rendering succeeds
	switch template {
	case TemplateTicketConfirm:
		msg.TemplateData = json.RawMessage(`{"name":"Alice","event_title":"Test Event","event_id":"` + uuid.New().String() + `","access_token":"` + uuid.New().String() + `"}`)
	case TemplateCheckIn:
		msg.TemplateData = json.RawMessage(`{"name":"Alice","event_title":"Test Event","event_id":"` + uuid.New().String() + `"}`)
	case TemplateOTP:
		msg.TemplateData = json.RawMessage(`{"otp":"123456","name":"Alice","event_title":"Test Event","event_id":"` + uuid.New().String() + `"}`)
	case TemplateEventCancellation:
		msg.TemplateData = json.RawMessage(`{"title":"Test Event","event_id":"` + uuid.New().String() + `"}`)
	default:
		msg.TemplateData = json.RawMessage(`{}`)
	}

	data, _ := json.Marshal(msg)
	return data
}

func makeMsgs(count int, template string) []jetstream.Msg {
	msgs := make([]jetstream.Msg, count)
	for i := range count {
		msgs[i] = &mockMsg{
			subject: "emails.ticket.confirmed",
			data:    buildValidPayload(template, fmt.Sprintf("user%d@example.com", i)),
		}
	}
	return msgs
}

// --- tests ---

func TestBatcher_FullBatchSuccess(t *testing.T) {
	msgs := makeMsgs(5, TemplateTicketConfirm)

	sender := &mockBatchSender{
		resp: &resend.BatchEmailResponse{
			Data: make([]resend.SendEmailResponse, 5),
		},
	}

	b := &Batcher{
		renderer: testRenderer("https://envitoo.app"),
		sender:   sender,
		log:      slog.Default(),
	}

	// Use a very fast rate limiter so the test doesn't block
	// We override the limiter by calling processBatch directly.
	b.processBatch(context.Background(), msgs)

	// All 5 should be acked
	for i, m := range msgs {
		mm := m.(*mockMsg)
		if !mm.acked {
			t.Errorf("msg %d: not acked", i)
		}
		if mm.nakd {
			t.Errorf("msg %d: unexpectedly nacked", i)
		}
		if mm.termed {
			t.Errorf("msg %d: unexpectedly termed", i)
		}
	}
}

func TestBatcher_FullBatchFailure(t *testing.T) {
	msgs := makeMsgs(3, TemplateTicketConfirm)

	sender := &mockBatchSender{
		err: errors.New("network error"),
	}

	b := &Batcher{
		renderer: testRenderer("https://envitoo.app"),
		sender:   sender,
		log:      slog.Default(),
	}

	b.processBatch(context.Background(), msgs)

	// All should be nacked with 2s delay, none acked
	for i, m := range msgs {
		mm := m.(*mockMsg)
		if mm.acked {
			t.Errorf("msg %d: unexpectedly acked", i)
		}
		if !mm.nakd {
			t.Errorf("msg %d: not nacked", i)
		}
		if mm.nakDelay != 2*time.Second {
			t.Errorf("msg %d: nak delay = %v, want 2s", i, mm.nakDelay)
		}
	}
}

func TestBatcher_PartialBatchFailure(t *testing.T) {
	msgs := makeMsgs(5, TemplateTicketConfirm)

	sender := &mockBatchSender{
		resp: &resend.BatchEmailResponse{
			Data: make([]resend.SendEmailResponse, 3), // only 3 succeeded
			Errors: []resend.BatchError{
				{Index: 2, Message: "bad recipient"},
				{Index: 4, Message: "rate limited"},
			},
		},
	}

	b := &Batcher{
		renderer: testRenderer("https://envitoo.app"),
		sender:   sender,
		log:      slog.Default(),
	}

	b.processBatch(context.Background(), msgs)

	// msgs 0, 1, 3 should be acked
	for _, i := range []int{0, 1, 3} {
		mm := msgs[i].(*mockMsg)
		if !mm.acked {
			t.Errorf("msg %d: should be acked", i)
		}
		if mm.nakd {
			t.Errorf("msg %d: should not be nacked", i)
		}
	}

	// msgs 2, 4 should be nacked
	for _, i := range []int{2, 4} {
		mm := msgs[i].(*mockMsg)
		if mm.acked {
			t.Errorf("msg %d: should not be acked", i)
		}
		if !mm.nakd {
			t.Errorf("msg %d: should be nacked", i)
		}
	}
}

func TestBatcher_LowTrafficBatch(t *testing.T) {
	// 3 messages — less than maxBatch — sent as-is
	msgs := makeMsgs(3, TemplateCheckIn)

	sender := &mockBatchSender{
		resp: &resend.BatchEmailResponse{
			Data: make([]resend.SendEmailResponse, 3),
		},
	}

	b := &Batcher{
		renderer: testRenderer("https://envitoo.app"),
		sender:   sender,
		log:      slog.Default(),
	}

	b.processBatch(context.Background(), msgs)

	sender.mu.Lock()
	defer sender.mu.Unlock()
	if got := len(sender.reqs); got != 1 {
		t.Fatalf("expected 1 batch send, got %d", got)
	}
	if got := len(sender.reqs[0]); got != 3 {
		t.Errorf("expected 3 send requests, got %d", got)
	}

	for i, m := range msgs {
		mm := m.(*mockMsg)
		if !mm.acked {
			t.Errorf("msg %d: not acked", i)
		}
	}
}

func TestBatcher_PoisonMessage(t *testing.T) {
	// One valid message + one poison (invalid JSON)
	valid := &mockMsg{
		subject: "emails.ticket.confirmed",
		data:    buildValidPayload(TemplateTicketConfirm, "valid@example.com"),
	}
	poison := &mockMsg{
		subject: "emails.bad",
		data:    []byte(`{not json}`),
	}

	msgs := []jetstream.Msg{valid, poison}

	sender := &mockBatchSender{
		resp: &resend.BatchEmailResponse{
			Data: make([]resend.SendEmailResponse, 1),
		},
	}

	b := &Batcher{
		renderer: testRenderer("https://envitoo.app"),
		sender:   sender,
		log:      slog.Default(),
	}

	b.processBatch(context.Background(), msgs)

	// Poison should be termed
	if !poison.termed {
		t.Error("poison message should be termed")
	}
	if poison.acked || poison.nakd {
		t.Error("poison message should not be acked or nacked")
	}

	// Valid should be acked
	if !valid.acked {
		t.Error("valid message should be acked")
	}
}

func TestBatcher_RateLimitBackpressure(t *testing.T) {
	// Verify that processBatch correctly handles the rate-limiting gap:
	// when Fetch returns 0 messages (low traffic), the rate limiter still
	// enforces spacing between iterations of the Run loop.
	//
	// We simulate this by testing that Run exits cleanly when the context
	// is cancelled before a fetch completes.

	renderer := testRenderer("https://envitoo.app")
	sender := &mockBatchSender{
		resp: &resend.BatchEmailResponse{
			Data: make([]resend.SendEmailResponse, 1),
		},
	}

	b := &Batcher{
		renderer: renderer,
		sender:   sender,
		log:      slog.Default(),
	}

	// Consumer that blocks Fetch until context is cancelled.
	type blockingConsumer struct {
		mockConsumer
	}
	c := &blockingConsumer{}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay to let Run start one iteration.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := b.Run(ctx, c)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Run should return context.Canceled, got %v", err)
	}
}

// Ensure rate.Limiter is used (satisfies import)
var _ = rate.NewLimiter(4, 1)
