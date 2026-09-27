-- name: ResolveMessagingConnection :one
-- First-active fallback only. A later router will score destination, network,
-- health, capacity, cost, and priority before selecting a connection.
SELECT * FROM messaging_connections
WHERE channel = sqlc.arg(channel)
  AND (
    (channel = 'whatsapp' AND scope = 'organization' AND organization_id = sqlc.arg(organization_id))
    OR (
      channel = 'sms'
      AND (
        (scope = 'organization' AND organization_id = sqlc.arg(organization_id))
        OR (scope = 'platform' AND organization_id IS NULL)
      )
    )
  )
  AND status = 'active'
ORDER BY
  CASE WHEN scope = 'organization' THEN 0 ELSE 1 END,
  created_at ASC
LIMIT 1;

-- name: ListActiveMessagingConnections :many
SELECT * FROM messaging_connections
WHERE status = 'active'
ORDER BY created_at ASC;

-- name: GetMessagingConnection :one
SELECT * FROM messaging_connections
WHERE id = sqlc.arg(id) AND status = 'active'
LIMIT 1;

-- name: ResolveInboundMessagingOrganization :one
SELECT COALESCE(connection.organization_id, number.organization_id)::uuid
FROM messaging_connections AS connection
LEFT JOIN phone_numbers AS number
  ON connection.scope = 'platform'
 AND number.number = sqlc.arg(to_address)
 AND number.status = 'active'
 AND number.sms_enabled = TRUE
WHERE connection.id = sqlc.arg(messaging_connection_id)
  AND connection.status = 'active'
  AND (
    (connection.scope = 'organization' AND connection.organization_id IS NOT NULL)
    OR (
      connection.channel = 'sms'
      AND connection.scope = 'platform'
      AND number.id IS NOT NULL
    )
  )
LIMIT 1;
