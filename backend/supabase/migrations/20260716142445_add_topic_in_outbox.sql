ALTER TABLE email_outbox ADD COLUMN topic TEXT NOT NULL DEFAULT 'emails.unknown';
