CREATE INDEX idx_email_outbox_published_cleanup
ON email_outbox (published_at)
WHERE status = 'PUBLISHED';