ALTER TABLE events
ADD COLUMN requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN cover_image_url TEXT,
ADD COLUMN calendar_id UUID REFERENCES calendars (id) ON DELETE CASCADE;

CREATE INDEX idx_events_calendar_id ON events (calendar_id);