-- name: CreateEmailOutbox :one
INSERT INTO email_outbox (recipient, template, template_data, topic)
VALUES ($1, $2, $3, $4)
RETURNING *;
