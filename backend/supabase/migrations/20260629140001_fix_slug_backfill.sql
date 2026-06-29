-- Backfill any rows that still have empty or NULL slugs (e.g. if the
-- original migration 20260625101904 failed on a multi-row database).
UPDATE events SET slug = gen_random_uuid()::text WHERE slug IS NULL OR slug = '';

-- Ensure the unique index exists.
CREATE UNIQUE INDEX IF NOT EXISTS idx_events_slug ON events (slug);
