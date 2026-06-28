CREATE TABLE volunteers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, user_id, role)
);

CREATE INDEX idx_volunteers_event_id ON volunteers (event_id);

CREATE INDEX idx_volunteers_user_id ON volunteers (user_id);