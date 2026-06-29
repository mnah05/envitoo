ALTER TABLE events
ADD COLUMN IF NOT EXISTS location_address TEXT;

-- Backfill existing in-person events so the updated constraint can apply.
UPDATE events
SET location_address = venue_name
WHERE
    is_virtual = FALSE
    AND venue_name IS NOT NULL
    AND location_address IS NULL;

ALTER TABLE events DROP CONSTRAINT IF EXISTS chk_physical_has_venue;

ALTER TABLE events ADD CONSTRAINT chk_physical_has_venue CHECK (
    is_virtual = TRUE
    OR (
        venue_name IS NOT NULL
        AND location_address IS NOT NULL
        AND lat IS NOT NULL
        AND lng IS NOT NULL
    )
);