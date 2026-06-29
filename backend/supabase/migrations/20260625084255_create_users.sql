-- Pulse - Authentication foundation
-- users table only. Schema source of truth; managed by Supabase migrations.

CREATE TABLE IF NOT EXISTS public.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_user_id  TEXT UNIQUE NOT NULL,
    email         TEXT UNIQUE NOT NULL,
    username      TEXT,
    avatar_url    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Hot path: auth middleware looks up users by Clerk user id (auth_user_id).
-- The UNIQUE constraint already creates an index; this is explicit documentation
-- of the access pattern for future readers and advisors.
CREATE INDEX IF NOT EXISTS idx_users_auth_user_id ON public.users (auth_user_id);
