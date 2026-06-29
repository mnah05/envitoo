CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    organizer_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    location TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    price_in_paise BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    status TEXT NOT NULL DEFAULT 'published' CHECK (
        status IN (
            'published',
            'cancelled',
            'completed'
        )
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_organizer_id ON events (organizer_id);

CREATE INDEX idx_events_starts_at ON events (starts_at);

CREATE INDEX idx_events_status ON events (status); 