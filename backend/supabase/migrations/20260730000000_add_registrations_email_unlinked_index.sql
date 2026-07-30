-- Partial index for fast lookup of orphaned guest registrations by email.
-- The LoadUser middleware checks SELECT EXISTS (...) with WHERE email = $1 AND user_id IS NULL
-- before attempting to link guest rows. This index makes that check cheap.
--
-- CONCURRENTLY is safe here only because psql (the Docker entrypoint runner)
-- does not wrap .sql files in a transaction block; each statement runs in its
-- own implicit transaction. If the project later adopts a migration tool that
-- wraps migrations in BEGIN/COMMIT, this statement will need to be split out
-- and run outside a transaction.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_registrations_email_unlinked
    ON registrations (email) WHERE user_id IS NULL;
