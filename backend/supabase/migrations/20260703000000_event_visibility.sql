CREATE TABLE event_visibility (
    event_id UUID PRIMARY KEY REFERENCES events (id) ON DELETE CASCADE,
    visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'unlisted'))
);

INSERT INTO event_visibility (event_id, visibility)
SELECT id, 'public' FROM events
ON CONFLICT (event_id) DO NOTHING;