-- name: FetchPendingOutboxEmails :many
SELECT id, recipient, template, template_data, topic, retries
FROM email_outbox
WHERE status = 'PENDING'
  AND retries < $2
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEmailsPublished :exec
UPDATE email_outbox
SET status = 'PUBLISHED', published_at = now()
WHERE id = ANY($1::uuid[]);

-- name: IncrementOutboxEmailRetries :exec
UPDATE email_outbox
SET retries = retries + 1,
    last_error = $2,
    status = CASE
        WHEN retries + 1 >= $3 THEN 'FAILED'::email_outbox_status
        ELSE status
    END,
    failed_at = CASE
        WHEN retries + 1 >= $3 THEN now()
        ELSE failed_at
    END
WHERE id = $1;

-- name: DeleteStaleOutboxEmails :execrows
DELETE FROM email_outbox
WHERE id IN (
    SELECT e.id
    FROM email_outbox e
    WHERE (
        e.status = 'PUBLISHED'
        AND e.published_at < $1
    ) OR (
        e.status = 'FAILED'
        AND COALESCE(e.failed_at, e.created_at) < $1
    )
    LIMIT $2
);
