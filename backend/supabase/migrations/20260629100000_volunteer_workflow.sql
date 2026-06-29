CREATE TABLE volunteer_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL UNIQUE REFERENCES events (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    max_applications INT4,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE volunteer_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, name)
);

CREATE INDEX idx_volunteer_roles_event_id ON volunteer_roles (event_id);

CREATE TABLE volunteer_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES volunteer_roles (id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'approved', 'rejected', 'cancelled')
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_volunteer_apps_event_user_active ON volunteer_applications (event_id, user_id)
WHERE
    status IN ('pending', 'approved');

CREATE INDEX idx_volunteer_apps_event_id ON volunteer_applications (event_id);

CREATE INDEX idx_volunteer_apps_user_id ON volunteer_applications (user_id);

CREATE INDEX idx_volunteer_apps_status ON volunteer_applications (status);