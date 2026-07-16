CREATE TYPE email_outbox_status AS ENUM (
    'PENDING',
    'PUBLISHED'
);
CREATE TABLE email_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    recipient TEXT NOT NULL,

    template TEXT NOT NULL,

    template_data JSONB NOT NULL DEFAULT '{}',

    status email_outbox_status NOT NULL DEFAULT 'PENDING',

    retries INT NOT NULL DEFAULT 0,
    last_error TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT now(),
    published_at TIMESTAMP
);


CREATE INDEX idx_email_outbox_pending
ON email_outbox(created_at)
WHERE status = 'PENDING';
