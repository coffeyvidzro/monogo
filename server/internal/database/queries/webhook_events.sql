-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (
    id,
    organization_id,
    event_type,
    object_type,
    object_id,
    payload,
    occurred_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(event_type),
    sqlc.arg(object_type),
    sqlc.narg(object_id),
    sqlc.arg(payload),
    sqlc.arg(occurred_at)
)
ON CONFLICT (id) DO NOTHING
RETURNING *;

-- name: GetWebhookEvent :one
SELECT webhook_events.*
FROM webhook_events
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_events.id = sqlc.arg(id)
  AND webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
LIMIT 1;

-- name: ListWebhookEvents :many
SELECT webhook_events.*
FROM webhook_events
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
ORDER BY webhook_events.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: ListWebhookEventsForObject :many
SELECT webhook_events.*
FROM webhook_events
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_events.organization_id = sqlc.arg(organization_id)
  AND webhook_events.object_type = sqlc.arg(object_type)
  AND webhook_events.object_id IS NOT DISTINCT FROM sqlc.narg(object_id)
  AND organizations.status = 'active'
ORDER BY webhook_events.occurred_at DESC, webhook_events.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: ListWebhookEventsFiltered :many
SELECT webhook_events.*
FROM webhook_events
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
  AND (sqlc.narg(event_type)::text IS NULL OR webhook_events.event_type = sqlc.narg(event_type))
  AND (sqlc.narg(object_type)::text IS NULL OR webhook_events.object_type = sqlc.narg(object_type))
  AND (sqlc.narg(object_id)::uuid IS NULL OR webhook_events.object_id = sqlc.narg(object_id))
  AND (sqlc.narg(before_occurred_at)::timestamptz IS NULL OR webhook_events.occurred_at < sqlc.narg(before_occurred_at))
ORDER BY webhook_events.occurred_at DESC, webhook_events.id DESC
LIMIT sqlc.arg(limit_count);
