-- Consolidate the volunteer_applications partial unique index after
-- migrations 20260629110000 and 20260629120000 both recreated it with
-- identical definitions.  This is a no-op when the index already exists
-- with the correct definition, but ensures a single canonical version.
DROP INDEX IF EXISTS idx_volunteer_apps_event_user_active;

CREATE UNIQUE INDEX idx_volunteer_apps_event_user_active
  ON volunteer_applications (event_id, user_id)
  WHERE status IN ('pending', 'approved');
