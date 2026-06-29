-- Follow-up for databases that applied 20260629100000_volunteer_workflow.sql
-- before cancelled status and the partial unique index were added.
--
-- Safe to run on fresh installs that already include the updated
-- 20260629100000 migration (operations are idempotent).

ALTER TABLE volunteer_applications
    DROP CONSTRAINT IF EXISTS volunteer_applications_event_id_user_id_key;

ALTER TABLE volunteer_applications
    DROP CONSTRAINT IF EXISTS volunteer_applications_status_check;

ALTER TABLE volunteer_applications
    ADD CONSTRAINT volunteer_applications_status_check
    CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'));

DROP INDEX IF EXISTS idx_volunteer_apps_event_user_active;

CREATE UNIQUE INDEX idx_volunteer_apps_event_user_active ON volunteer_applications (event_id, user_id)
WHERE
    status IN ('pending', 'approved');