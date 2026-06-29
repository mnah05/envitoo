CREATE INDEX IF NOT EXISTS idx_payments_event_id ON payments (event_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments (user_id);
CREATE INDEX IF NOT EXISTS idx_registrations_email ON registrations (email);
CREATE INDEX IF NOT EXISTS idx_volunteer_applications_role_id ON volunteer_applications (role_id);
CREATE INDEX IF NOT EXISTS idx_registrations_event_status ON registrations (event_id, status);
