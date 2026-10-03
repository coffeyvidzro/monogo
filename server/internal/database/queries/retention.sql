-- name: ListRetentionPolicies :many
SELECT *
FROM retention_policies
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY resource;

-- name: GetRetentionPolicy :one
SELECT *
FROM retention_policies
WHERE organization_id = sqlc.arg(organization_id)
  AND resource = sqlc.arg(resource)
LIMIT 1;

-- name: UpsertRetentionPolicy :one
INSERT INTO retention_policies (
    organization_id,
    resource,
    retention_days,
    enabled
)
VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(resource),
    sqlc.arg(retention_days),
    sqlc.arg(enabled)
)
ON CONFLICT (organization_id, resource)
DO UPDATE SET
    retention_days = EXCLUDED.retention_days,
    enabled = EXCLUDED.enabled
RETURNING *;

-- name: DeleteRetentionPolicy :exec
DELETE FROM retention_policies
WHERE organization_id = sqlc.arg(organization_id)
  AND resource = sqlc.arg(resource);

-- name: ListEnabledRetentionPolicies :many
SELECT *
FROM retention_policies
WHERE enabled
ORDER BY organization_id, resource;

-- name: ListExpiredRecordingsForRetention :many
SELECT id
FROM recordings
WHERE organization_id = sqlc.arg(organization_id)
  AND status = 'completed'
  AND created_at < sqlc.arg(created_before)
ORDER BY created_at
LIMIT sqlc.arg(batch_size);

-- name: ListExpiredConversationsForRetention :many
SELECT id
FROM voice_agent_sessions
WHERE organization_id = sqlc.arg(organization_id)
  AND state IN ('completed', 'failed', 'cancelled')
  AND ended_at IS NOT NULL
  AND ended_at < sqlc.arg(created_before)
ORDER BY ended_at
LIMIT sqlc.arg(batch_size);

-- name: DeleteConversationForRetention :exec
DELETE FROM voice_agent_sessions
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('completed', 'failed', 'cancelled');

-- name: ListExpiredAuditEventsForRetention :many
SELECT id
FROM audit_events
WHERE organization_id = sqlc.arg(organization_id)
  AND occurred_at < sqlc.arg(created_before)
ORDER BY occurred_at
LIMIT sqlc.arg(batch_size);

-- name: DeleteAuditEventForRetention :exec
DELETE FROM audit_events
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id);
