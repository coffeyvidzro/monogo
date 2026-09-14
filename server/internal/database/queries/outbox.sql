-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (
    subject,
    aggregate_type,
    aggregate_id,
    payload,
    headers,
    available_at
) VALUES (
    sqlc.arg(subject),
    sqlc.arg(aggregate_type),
    sqlc.arg(aggregate_id),
    sqlc.arg(payload)::jsonb,
    COALESCE(sqlc.narg(headers)::jsonb, '{}'::jsonb),
    COALESCE(sqlc.narg(available_at), NOW())
)
RETURNING *;

-- name: ClaimPendingEvents :many
UPDATE outbox_events
SET
    locked_at = NOW(),
    locked_by = sqlc.arg(locked_by),
    attempts = attempts + 1,
    updated_at = NOW()
WHERE id IN (
    SELECT id
    FROM outbox_events
    WHERE published_at IS NULL
    AND locked_at IS NULL
    AND available_at <= NOW()
    ORDER BY available_at, created_at
    LIMIT sqlc.arg(batch_size)::integer
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: MarkEventPublished :exec
UPDATE outbox_events
SET
    published_at = NOW(),
    locked_at = NULL,
    locked_by = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND published_at IS NULL;

-- name: MarkEventFailed :exec
UPDATE outbox_events
SET
    locked_at = NULL,
    locked_by = NULL,
    last_error = sqlc.narg(last_error),
    available_at = NOW() + (sqlc.arg(retry_delay_seconds)::integer * INTERVAL '1 second'),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND published_at IS NULL;

-- name: DeleteOldPublishedEvents :exec
DELETE FROM outbox_events
WHERE published_at IS NOT NULL
AND published_at < NOW() - (sqlc.arg(retention_days)::integer * INTERVAL '1 day');

-- name: ReleaseStaleLocks :exec
UPDATE outbox_events
SET
    locked_at = NULL,
    locked_by = NULL,
    updated_at = NOW()
WHERE published_at IS NULL
AND locked_at IS NOT NULL
AND locked_at < NOW() - (sqlc.arg(lock_timeout_seconds)::integer * INTERVAL '1 second');

-- name: MarkEventProcessed :exec
INSERT INTO processed_events (
    consumer_name,
    event_id,
    metadata
) VALUES (
    sqlc.arg(consumer_name),
    sqlc.arg(event_id),
    COALESCE(sqlc.narg(metadata)::jsonb, '{}'::jsonb)
)
ON CONFLICT (consumer_name, event_id) DO NOTHING;

-- name: IsEventProcessed :one
SELECT EXISTS (
    SELECT 1
    FROM processed_events
    WHERE consumer_name = sqlc.arg(consumer_name)
    AND event_id = sqlc.arg(event_id)
) AS processed;

-- name: ListPendingEvents :many
SELECT *
FROM outbox_events
WHERE published_at IS NULL
AND locked_at IS NULL
AND available_at <= NOW()
ORDER BY available_at, created_at
LIMIT sqlc.arg(limit_count)::integer;

-- name: GetEventByID :one
SELECT *
FROM outbox_events
WHERE id = sqlc.arg(id)
LIMIT 1;
