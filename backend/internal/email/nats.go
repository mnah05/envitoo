package email

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	// StreamName is the JetStream stream for transactional email subjects.
	StreamName = "EMAILS"
	// StreamSubjects captures every emails.* topic produced by the API.
	StreamSubjects = "emails.>"
	// ConsumerName is the durable consumer used by cmd/mailer.
	ConsumerName = "email-resend"
)

// EnsureEmailsStream creates or updates the EMAILS stream.
// Safe to call from both the outbox drainer and the mailer on startup.
func EnsureEmailsStream(ctx context.Context, js jetstream.JetStream) (jetstream.Stream, error) {
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{StreamSubjects},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Discard:   jetstream.DiscardOld,
		MaxAge:    7 * 24 * time.Hour,
		Duplicates: time.Minute, // aligns with MsgID dedup on PublishAsync
	})
	if err != nil {
		return nil, fmt.Errorf("ensure EMAILS stream: %w", err)
	}
	return stream, nil
}

// EnsureMailerConsumer creates or updates the durable mailer consumer on EMAILS.
func EnsureMailerConsumer(ctx context.Context, js jetstream.JetStream) (jetstream.Consumer, error) {
	if _, err := EnsureEmailsStream(ctx, js); err != nil {
		return nil, err
	}
	consumer, err := js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxDeliver:    10,
		FilterSubject: StreamSubjects,
		// DeliverAll so messages published before the mailer starts are not lost.
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure %s consumer: %w", ConsumerName, err)
	}
	return consumer, nil
}
