CREATE INDEX idx_email_outbox_failed_cleanup
ON email_outbox (failed_at)
WHERE status = 'FAILED';