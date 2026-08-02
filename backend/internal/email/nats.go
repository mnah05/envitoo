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
//
// Note 1: JetStream is NATS' persistent messaging layer. Unlike core NATS
// (at-most-once), JetStream provides at-least-once delivery with persistence
// to disk. If the mailer crashes, messages survive restart.
//
// Note 2: WorkQueuePolicy means each message is delivered to exactly one
// consumer. This prevents the mailer from sending duplicate emails if
// multiple mailer instances are running for high availability.
//
// Note 3: Duplicates dedup window is set to 1 minute. If a message with the
// same MsgID is published within that window, JetStream silently drops the
// duplicate. This aligns with the PublishAsync MsgId used by the drainer.
func EnsureEmailsStream(ctx context.Context, js jetstream.JetStream) (jetstream.Stream, error) {
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       StreamName,
		Subjects:   []string{StreamSubjects},
		Retention:  jetstream.WorkQueuePolicy,
		Storage:    jetstream.FileStorage,
		Discard:    jetstream.DiscardOld,
		MaxAge:     7 * 24 * time.Hour,
		Duplicates: time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure EMAILS stream: %w", err)
	}
	return stream, nil
}

// EnsureMailerConsumer creates or updates the durable pull-based mailer
// consumer on EMAILS.
//
// The consumer is configured for pull-based Fetch (used by Batcher) rather
// than push-based Consume. MaxAckPending is 2× the batch size so the broker
// keeps enough messages available for efficient batching without flooding
// memory. InactiveThreshold is 5 minutes so the consumer is not cleaned up
// during extended idle periods.
func EnsureMailerConsumer(ctx context.Context, js jetstream.JetStream) (jetstream.Consumer, error) {
	if _, err := EnsureEmailsStream(ctx, js); err != nil {
		return nil, err
	}
	consumer, err := js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:           ConsumerName,
		AckPolicy:         jetstream.AckExplicitPolicy,
		AckWait:           60 * time.Second,
		MaxDeliver:        10,
		MaxAckPending:     200,
		MaxRequestBatch:   100,
		FilterSubject:     StreamSubjects,
		DeliverPolicy:     jetstream.DeliverAllPolicy,
		InactiveThreshold: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure %s consumer: %w", ConsumerName, err)
	}
	return consumer, nil
}
