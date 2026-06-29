CREATE TABLE registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'approved' CHECK (
        status IN (
            'pending',
            'approved',
            'rejected',
            'cancelled'
        )
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, email)
);

CREATE INDEX idx_registrations_event_id ON registrations (event_id);

CREATE INDEX idx_registrations_user_id ON registrations (user_id);

CREATE INDEX idx_registrations_status ON registrations (status);

CREATE UNIQUE INDEX idx_registrations_user_event ON registrations (event_id, user_id)
WHERE
    user_id IS NOT NULL;