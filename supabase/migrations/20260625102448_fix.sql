ALTER TABLE events
ADD COLUMN category TEXT CHECK (
    category IN (
        'technical',
        'cultural',
        'sports',
        'workshop',
        'seminar',
        'hackathon',
        'other'
    )
);

CREATE INDEX idx_events_category ON events (category);

ALTER TABLE events ALTER COLUMN slug SET NOT NULL;

ALTER TABLE payments
ADD CONSTRAINT uq_payments_registration_id UNIQUE (registration_id);