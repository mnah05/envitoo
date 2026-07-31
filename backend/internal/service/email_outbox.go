package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mnah05/envitoo/internal/db"
	"github.com/mnah05/envitoo/internal/email"
)

// enqueueEmail marshals an email payload into the email_outbox table as
// part of the caller's transaction. The row is picked up by the outbox
// drainer worker (cmd/worker), published to NATS JetStream, and finally
// rendered and sent by the mailer (cmd/mailer).
//
// The template name and NATS topic come from the EmailData value itself,
// so a payload can never be enqueued under a mismatched template/topic pair.
func enqueueEmail(ctx context.Context, qtx *db.Queries, recipient string, data email.EmailData) error {
	templateData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal template data: %w", err)
	}
	if _, err := qtx.CreateEmailOutbox(ctx, db.CreateEmailOutboxParams{
		Recipient:    recipient,
		Template:     data.Template(),
		TemplateData: templateData,
		Topic:        data.Topic(),
	}); err != nil {
		return fmt.Errorf("create email outbox: %w", err)
	}
	return nil
}
