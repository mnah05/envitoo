-- Allow re-registration after cancel/reject by scoping uniqueness to
-- active statuses only (matches volunteer_applications pattern).
-- Cancelled/rejected rows no longer block a new pending/approved row.

ALTER TABLE registrations DROP CONSTRAINT IF EXISTS registrations_event_id_email_key;

CREATE UNIQUE INDEX idx_registrations_event_email_active
  ON registrations (event_id, email)
  WHERE status IN ('pending', 'approved');

DROP INDEX IF EXISTS idx_registrations_user_event;

CREATE UNIQUE INDEX idx_registrations_user_event
  ON registrations (event_id, user_id)
  WHERE user_id IS NOT NULL
    AND status IN ('pending', 'approved');
