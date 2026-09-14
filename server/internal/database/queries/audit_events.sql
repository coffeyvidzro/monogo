-- name: InsertAuditEvent :exec
INSERT INTO audit_events (
    organization_id,
    actor_type,
    actor_id,
    action,
    target_type,
    target_id,
    metadata
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(actor_type),
    sqlc.arg(actor_id),
    sqlc.arg(action),
    sqlc.arg(target_type),
    sqlc.arg(target_id),
    sqlc.arg(metadata)::jsonb
);

-- name: ListAuditEventsByOrganizationID :many
SELECT
    id,
    organization_id,
    actor_type,
    actor_id,
    action,
    target_type,
    target_id,
    metadata,
    occurred_at
FROM audit_events
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(limit_count)::integer
OFFSET sqlc.arg(offset_count)::integer;
