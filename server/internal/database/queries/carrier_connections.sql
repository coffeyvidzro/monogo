-- name: CreateCarrierConnection :one
INSERT INTO carrier_connections (
    organization_id,
    provider_id,
    scope,
    name,
    status,
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext,
    inbound_enabled,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext,
    max_cps,
    max_concurrent_calls,
    max_daily_minutes,
    codecs,
    supports_video,
    supports_fax
)
SELECT
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(provider_id) AS provider_id,
    'organization' AS scope,
    sqlc.arg(name) AS name,
    COALESCE(sqlc.narg(status), 'active') AS status,
    COALESCE(sqlc.narg(outbound_auth_method), 'none') AS outbound_auth_method,
    sqlc.narg(auth_username) AS auth_username,
    sqlc.narg(auth_secret_ciphertext) AS auth_secret_ciphertext,
    COALESCE(sqlc.narg(inbound_enabled), false) AS inbound_enabled,
    COALESCE(sqlc.narg(inbound_auth_method), 'ip') AS inbound_auth_method,
    sqlc.narg(inbound_username) AS inbound_username,
    sqlc.narg(inbound_secret_ciphertext) AS inbound_secret_ciphertext,
    COALESCE(sqlc.narg(max_cps), 10) AS max_cps,
    COALESCE(sqlc.narg(max_concurrent_calls), 100) AS max_concurrent_calls,
    sqlc.narg(max_daily_minutes) AS max_daily_minutes,
    COALESCE(sqlc.narg(codecs), ARRAY['PCMU','PCMA']::TEXT[]) AS codecs,
    COALESCE(sqlc.narg(supports_video), false) AS supports_video,
    COALESCE(sqlc.narg(supports_fax), false) AS supports_fax
FROM organizations AS o
JOIN carrier_providers AS cp ON cp.id = sqlc.arg(provider_id)
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
  AND cp.status = 'active'
RETURNING *;

-- name: CreatePlatformCarrierConnection :one
INSERT INTO carrier_connections (
    organization_id,
    provider_id,
    scope,
    name,
    status,
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext,
    inbound_enabled,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext,
    max_cps,
    max_concurrent_calls,
    max_daily_minutes,
    codecs,
    supports_video,
    supports_fax
)
SELECT
    NULL::UUID AS organization_id,
    sqlc.arg(provider_id) AS provider_id,
    'platform' AS scope,
    sqlc.arg(name) AS name,
    COALESCE(sqlc.narg(status), 'active') AS status,
    COALESCE(sqlc.narg(outbound_auth_method), 'none') AS outbound_auth_method,
    sqlc.narg(auth_username) AS auth_username,
    sqlc.narg(auth_secret_ciphertext) AS auth_secret_ciphertext,
    COALESCE(sqlc.narg(inbound_enabled), false) AS inbound_enabled,
    COALESCE(sqlc.narg(inbound_auth_method), 'ip') AS inbound_auth_method,
    sqlc.narg(inbound_username) AS inbound_username,
    sqlc.narg(inbound_secret_ciphertext) AS inbound_secret_ciphertext,
    COALESCE(sqlc.narg(max_cps), 10) AS max_cps,
    COALESCE(sqlc.narg(max_concurrent_calls), 100) AS max_concurrent_calls,
    sqlc.narg(max_daily_minutes) AS max_daily_minutes,
    COALESCE(sqlc.narg(codecs), ARRAY['PCMU','PCMA']::TEXT[]) AS codecs,
    COALESCE(sqlc.narg(supports_video), false) AS supports_video,
    COALESCE(sqlc.narg(supports_fax), false) AS supports_fax
FROM carrier_providers AS cp
WHERE cp.id = sqlc.arg(provider_id)
  AND cp.status = 'active'
RETURNING *;

-- name: GetCarrierConnectionByID :one
SELECT
    id,
    organization_id,
    provider_id,
    scope,
    name,
    status,
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext IS NOT NULL AS has_outbound_credentials,
    inbound_enabled,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext IS NOT NULL AS has_inbound_credentials,
    max_cps,
    max_concurrent_calls,
    max_daily_minutes,
    codecs,
    supports_video,
    supports_fax,
    created_at,
    updated_at
FROM carrier_connections
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetPlatformCarrierConnectionByID :one
SELECT *
FROM carrier_connections
WHERE id = sqlc.arg(id)
  AND scope = 'platform'
  AND organization_id IS NULL
LIMIT 1;

-- name: ListCarrierConnectionsByOrganizationID :many
SELECT
    id,
    organization_id,
    provider_id,
    scope,
    name,
    status,
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext IS NOT NULL AS has_outbound_credentials,
    inbound_enabled,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext IS NOT NULL AS has_inbound_credentials,
    max_cps,
    max_concurrent_calls,
    max_daily_minutes,
    codecs,
    supports_video,
    supports_fax,
    created_at,
    updated_at
FROM carrier_connections
WHERE scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: ListActiveCarrierConnectionsByOrganizationID :many
SELECT
    id,
    organization_id,
    provider_id,
    scope,
    name,
    status,
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext IS NOT NULL AS has_outbound_credentials,
    inbound_enabled,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext IS NOT NULL AS has_inbound_credentials,
    max_cps,
    max_concurrent_calls,
    max_daily_minutes,
    codecs,
    supports_video,
    supports_fax,
    created_at,
    updated_at
FROM carrier_connections
WHERE scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
ORDER BY created_at DESC;

-- name: ListPlatformCarrierConnections :many
SELECT *
FROM carrier_connections
WHERE scope = 'platform'
  AND organization_id IS NULL
ORDER BY created_at DESC;

-- name: ListActivePlatformCarrierConnections :many
SELECT *
FROM carrier_connections
WHERE scope = 'platform'
  AND organization_id IS NULL
  AND status = 'active'
ORDER BY created_at DESC;

-- name: ListProviderRoutingTargets :many
SELECT
    cc.id AS carrier_connection_id,
    resource.provider_resource_id
FROM carrier_connections AS cc
JOIN carrier_connection_provider_resources AS resource
  ON resource.carrier_connection_id = cc.id
 AND resource.provider_id = cc.provider_id
 AND resource.resource_type = 'voice_in_trunk'
WHERE cc.provider_id = sqlc.arg(provider_id)
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
  AND cc.status = 'active'
  AND cc.inbound_enabled = true
ORDER BY cc.created_at ASC
LIMIT 2;

-- name: GetCarrierConnectionCredentials :one
SELECT
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext
FROM carrier_connections
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
LIMIT 1;

-- name: GetPlatformCarrierConnectionCredentials :one
SELECT
    outbound_auth_method,
    auth_username,
    auth_secret_ciphertext,
    inbound_auth_method,
    inbound_username,
    inbound_secret_ciphertext
FROM carrier_connections
WHERE id = sqlc.arg(id)
  AND scope = 'platform'
  AND organization_id IS NULL
  AND status = 'active'
LIMIT 1;

-- name: UpdateCarrierConnection :one
UPDATE carrier_connections
SET
    name = COALESCE(sqlc.narg(name), name),
    status = COALESCE(sqlc.narg(status), status),
    inbound_enabled = COALESCE(sqlc.narg(inbound_enabled), inbound_enabled),
    max_cps = COALESCE(sqlc.narg(max_cps), max_cps),
    max_concurrent_calls = COALESCE(sqlc.narg(max_concurrent_calls), max_concurrent_calls),
    max_daily_minutes = COALESCE(sqlc.narg(max_daily_minutes), max_daily_minutes),
    codecs = COALESCE(sqlc.narg(codecs), codecs),
    supports_video = COALESCE(sqlc.narg(supports_video), supports_video),
    supports_fax = COALESCE(sqlc.narg(supports_fax), supports_fax),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: UpdatePlatformCarrierConnection :one
UPDATE carrier_connections
SET
    name = COALESCE(sqlc.narg(name), name),
    status = COALESCE(sqlc.narg(status), status),
    inbound_enabled = COALESCE(sqlc.narg(inbound_enabled), inbound_enabled),
    max_cps = COALESCE(sqlc.narg(max_cps), max_cps),
    max_concurrent_calls = COALESCE(sqlc.narg(max_concurrent_calls), max_concurrent_calls),
    max_daily_minutes = COALESCE(sqlc.narg(max_daily_minutes), max_daily_minutes),
    codecs = COALESCE(sqlc.narg(codecs), codecs),
    supports_video = COALESCE(sqlc.narg(supports_video), supports_video),
    supports_fax = COALESCE(sqlc.narg(supports_fax), supports_fax),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'platform'
  AND organization_id IS NULL
RETURNING *;

-- name: SetCarrierConnectionOutboundDigestAuth :exec
UPDATE carrier_connections
SET
    outbound_auth_method = 'digest',
    auth_username = sqlc.arg(auth_username),
    auth_secret_ciphertext = sqlc.arg(auth_secret_ciphertext),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id);

-- name: ClearCarrierConnectionOutboundAuth :exec
UPDATE carrier_connections
SET
    outbound_auth_method = 'none',
    auth_username = NULL,
    auth_secret_ciphertext = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id);

-- name: SetCarrierConnectionInboundDigestAuth :exec
UPDATE carrier_connections
SET
    inbound_auth_method = 'digest',
    inbound_username = sqlc.arg(inbound_username),
    inbound_secret_ciphertext = sqlc.arg(inbound_secret_ciphertext),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id);

-- name: SetCarrierConnectionInboundIPAuth :exec
UPDATE carrier_connections
SET
    inbound_auth_method = 'ip',
    inbound_username = NULL,
    inbound_secret_ciphertext = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id);

-- name: SetCarrierConnectionInboundNoAuth :exec
UPDATE carrier_connections
SET
    inbound_auth_method = 'none',
    inbound_username = NULL,
    inbound_secret_ciphertext = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id);

-- name: DisableCarrierConnection :exec
UPDATE carrier_connections
SET
    status = 'disabled',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active';

-- name: EnableCarrierConnection :exec
UPDATE carrier_connections
SET
    status = 'active',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND scope = 'organization'
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'disabled';

-- name: CreateCarrierConnectionSourceIP :one
INSERT INTO carrier_connection_source_ips (
    organization_id,
    carrier_connection_id,
    cidr
)
SELECT
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(carrier_connection_id) AS carrier_connection_id,
    sqlc.arg(cidr) AS cidr
FROM carrier_connections AS cc
WHERE cc.id = sqlc.arg(carrier_connection_id)
  AND cc.scope = 'organization'
  AND cc.organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: ListCarrierConnectionSourceIPs :many
SELECT src.*
FROM carrier_connection_source_ips AS src
JOIN carrier_connections AS cc
  ON cc.id = src.carrier_connection_id
 AND cc.organization_id = src.organization_id
WHERE src.carrier_connection_id = sqlc.arg(carrier_connection_id)
  AND src.organization_id = sqlc.arg(organization_id)
  AND cc.scope = 'organization'
ORDER BY src.cidr ASC;

-- name: DeleteCarrierConnectionSourceIP :exec
DELETE FROM carrier_connection_source_ips AS src
USING carrier_connections AS cc
WHERE src.id = sqlc.arg(id)
  AND src.carrier_connection_id = sqlc.arg(carrier_connection_id)
  AND src.organization_id = sqlc.arg(organization_id)
  AND cc.id = src.carrier_connection_id
  AND cc.scope = 'organization'
  AND cc.organization_id = src.organization_id;

-- name: ResolveCarrierConnectionBySourceIP :one
SELECT cc.*
FROM carrier_connection_source_ips AS src
JOIN carrier_connections AS cc
  ON cc.id = src.carrier_connection_id
 AND cc.organization_id = src.organization_id
WHERE sqlc.arg(source_ip)::INET <<= src.cidr
  AND cc.scope = 'organization'
  AND cc.status = 'active'
  AND cc.inbound_enabled = true
  AND cc.inbound_auth_method = 'ip'
ORDER BY masklen(src.cidr) DESC
LIMIT 1;

-- name: ListBackofficeCarrierConnections :many
SELECT
    cc.id::TEXT AS id,
    CAST(COALESCE(cc.organization_id::TEXT, '—') AS TEXT) AS organization_id,
    COALESCE(o.name, 'Platform') AS organization_name,
    cc.name,
    cp.name AS provider_name,
    cc.scope,
    cc.status,
    cc.inbound_enabled,
    cc.max_cps,
    cc.max_concurrent_calls,
    COUNT(t.id)::BIGINT AS trunk_count
FROM carrier_connections AS cc
JOIN carrier_providers AS cp ON cp.id = cc.provider_id
LEFT JOIN organizations AS o ON o.id = cc.organization_id
LEFT JOIN trunks AS t ON t.carrier_connection_id = cc.id
GROUP BY
    cc.id,
    o.name,
    cc.name,
    cp.name,
    cc.scope,
    cc.status,
    cc.inbound_enabled,
    cc.max_cps,
    cc.max_concurrent_calls,
    cc.created_at
ORDER BY cc.created_at DESC
LIMIT 100;

-- name: GetBackofficeCarrierConnection :one
SELECT
    cc.id::TEXT AS id,
    CAST(COALESCE(cc.organization_id::TEXT, '—') AS TEXT) AS organization_id,
    COALESCE(o.name, 'Platform') AS organization_name,
    cc.provider_id::TEXT AS provider_id,
    cp.name AS provider_name,
    cc.name,
    cc.scope,
    cc.status,
    cc.outbound_auth_method,
    cc.inbound_enabled,
    cc.inbound_auth_method,
    cc.max_cps,
    cc.max_concurrent_calls,
    CAST(COALESCE(cc.max_daily_minutes::TEXT, '—') AS TEXT) AS max_daily_minutes,
    array_to_string(cc.codecs, ', ') AS codecs,
    cc.supports_video,
    cc.supports_fax,
    COUNT(DISTINCT t.id)::BIGINT AS trunk_count,
    COUNT(DISTINCT src.id)::BIGINT AS source_ip_count,
    to_char(cc.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
    to_char(cc.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM carrier_connections AS cc
JOIN carrier_providers AS cp ON cp.id = cc.provider_id
LEFT JOIN organizations AS o ON o.id = cc.organization_id
LEFT JOIN trunks AS t ON t.carrier_connection_id = cc.id
LEFT JOIN carrier_connection_source_ips AS src ON src.carrier_connection_id = cc.id
WHERE cc.id = sqlc.arg(id)
GROUP BY cc.id, o.name, cp.name
LIMIT 1;

-- name: ListBackofficeCarrierConnectionSourceIPs :many
SELECT src.id::TEXT AS id, src.cidr::TEXT AS cidr,
       to_char(src.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at
FROM carrier_connection_source_ips AS src
WHERE src.carrier_connection_id = sqlc.arg(carrier_connection_id)
ORDER BY src.created_at;

-- name: ListBackofficeCarrierConnectionResources :many
SELECT resource_type, provider_resource_id,
       to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
       to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM carrier_connection_provider_resources
WHERE carrier_connection_id = sqlc.arg(carrier_connection_id)
ORDER BY resource_type;
