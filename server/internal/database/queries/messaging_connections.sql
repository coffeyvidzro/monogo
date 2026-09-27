-- name: ResolveMessagingConnection :one
SELECT * FROM messaging_connections
WHERE organization_id = sqlc.arg(organization_id)
  AND channel = sqlc.arg(channel)
  AND status = 'active'
ORDER BY created_at ASC
LIMIT 1;

-- name: ListActiveMessagingConnections :many
SELECT * FROM messaging_connections
WHERE status = 'active'
ORDER BY created_at ASC;

-- name: GetMessagingConnection :one
SELECT * FROM messaging_connections
WHERE id = sqlc.arg(id) AND status = 'active'
LIMIT 1;
