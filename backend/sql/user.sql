-- name: GetUserByAuthUserID :one
SELECT * FROM users WHERE auth_user_id = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
-- Inserts a new local user row on first login. ON CONFLICT DO NOTHING
-- makes this safe against concurrent first-logins and user.created
-- webhooks racing on the same auth_user_id. When the conflict fires,
-- no row is returned (pgx.ErrNoRows); callers must re-fetch with
-- GetUserByAuthUserID to pick up the winning row.
INSERT INTO users (auth_user_id, email, username, avatar_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (auth_user_id) DO NOTHING
RETURNING *;
