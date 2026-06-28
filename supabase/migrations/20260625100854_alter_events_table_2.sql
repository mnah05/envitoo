ALTER TABLE events
DROP COLUMN location,
ADD COLUMN is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN meeting_url TEXT,
ADD COLUMN venue_name TEXT,
ADD COLUMN lat NUMERIC(9, 6),
ADD COLUMN lng NUMERIC(9, 6),
ADD CONSTRAINT chk_virtual_has_url CHECK (
    is_virtual = FALSE
    OR meeting_url IS NOT NULL
),
ADD CONSTRAINT chk_physical_has_venue CHECK (
    is_virtual = TRUE
    OR (
        venue_name IS NOT NULL
        AND lat IS NOT NULL
        AND lng IS NOT NULL
    )
);