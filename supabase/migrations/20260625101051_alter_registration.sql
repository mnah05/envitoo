ALTER TABLE registrations
ADD COLUMN access_token UUID NOT NULL DEFAULT gen_random_uuid(),
ADD COLUMN token_expires_at TIMESTAMPTZ, -- NULL = never expires
ADD COLUMN checked_in_at TIMESTAMPTZ, -- NULL = not checked in yet
ADD CONSTRAINT uq_access_token UNIQUE (access_token);

CREATE INDEX idx_registrations_access_token ON registrations (access_token);
