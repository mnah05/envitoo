-- Allow a user to re-apply after being rejected.
--
-- Idempotent fix for databases that still have the older partial unique
-- index (WHERE status <> 'cancelled'). Fresh installs and migration
-- 20260629110000 already use status IN ('pending', 'approved').

DROP INDEX IF EXISTS idx_volunteer_apps_event_user_active;

CREATE UNIQUE INDEX IF NOT EXISTS idx_volunteer_apps_event_user_active
  ON volunteer_applications (event_id, user_id)
  WHERE status IN ('pending', 'approved');
