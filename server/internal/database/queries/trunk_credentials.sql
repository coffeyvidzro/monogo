-- name: CreateManagedTrunk :one
INSERT INTO trunks (
    organization_id,
    carrier_connection_id,
    provisioning_mode,
    name,
    direction,
    status,
    managed_default
)
VALUES (
    sqlc.arg(organization_id),
    NULL,
    'managed',
    sqlc.arg(name),
    COALESCE(sqlc.narg(direction), 'bidirectional'),
    COALESCE(sqlc.narg(status), 'active'),
    false
)
RETURNING *;

-- name: CreateTrunkCredential :one
INSERT INTO trunk_credentials (
    trunk_id,
    organization_id,
    username,
    realm,
    ha1_md5
)
SELECT
    t.id,
    t.organization_id,
    sqlc.arg(username),
    sqlc.arg(realm),
    sqlc.arg(ha1_md5)
FROM trunks AS t
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND t.provisioning_mode = 'managed'
  AND t.carrier_connection_id IS NULL
RETURNING *;

-- name: RotateTrunkCredential :one
UPDATE trunk_credentials AS tc
SET
    username = sqlc.arg(username),
    realm = sqlc.arg(realm),
    ha1_md5 = sqlc.arg(ha1_md5),
    updated_at = NOW()
FROM trunks AS t
WHERE tc.trunk_id = sqlc.arg(trunk_id)
  AND tc.organization_id = sqlc.arg(organization_id)
  AND t.id = tc.trunk_id
  AND t.organization_id = tc.organization_id
  AND t.provisioning_mode = 'managed'
  AND t.carrier_connection_id IS NULL
RETURNING tc.*;

-- name: GetTrunkCredential :one
SELECT tc.*
FROM trunk_credentials AS tc
JOIN trunks AS t ON t.id = tc.trunk_id
WHERE tc.trunk_id = sqlc.arg(trunk_id)
  AND tc.organization_id = sqlc.arg(organization_id)
  AND t.organization_id = tc.organization_id
  AND t.provisioning_mode = 'managed'
  AND t.carrier_connection_id IS NULL
LIMIT 1;
