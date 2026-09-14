-- name: CreateTrunk :one
INSERT INTO trunks (
    organization_id,
    carrier_connection_id,
    provisioning_mode,
    name,
    direction,
    status,
    managed_default
)
SELECT
    cc.organization_id AS organization_id,
    cc.id AS carrier_connection_id,
    sqlc.arg(provisioning_mode) AS provisioning_mode,
    sqlc.arg(name) AS name,
    COALESCE(sqlc.narg(direction), 'bidirectional') AS direction,
    COALESCE(sqlc.narg(status), 'active') AS status,
    false AS managed_default
FROM carrier_connections AS cc
JOIN carrier_providers AS cp ON cp.id = cc.provider_id
WHERE cc.id = sqlc.arg(carrier_connection_id)
  AND cc.scope = 'organization'
  AND cc.organization_id = sqlc.arg(organization_id)
  AND cc.status = 'active'
  AND (
      (sqlc.arg(provisioning_mode)::TEXT = 'byoc' AND cp.slug <> 'leamout')
      OR (sqlc.arg(provisioning_mode)::TEXT = 'managed' AND cp.slug = 'leamout')
  )
RETURNING *;

-- name: CreatePlatformTrunk :one
INSERT INTO trunks (
    organization_id,
    carrier_connection_id,
    provisioning_mode,
    name,
    direction,
    status,
    managed_default
)
SELECT
    NULL::UUID AS organization_id,
    cc.id AS carrier_connection_id,
    'managed' AS provisioning_mode,
    sqlc.arg(name) AS name,
    COALESCE(sqlc.narg(direction), 'bidirectional') AS direction,
    COALESCE(sqlc.narg(status), 'active') AS status,
    COALESCE(sqlc.narg(managed_default), false) AS managed_default
FROM carrier_connections AS cc
WHERE cc.id = sqlc.arg(carrier_connection_id)
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
  AND cc.status = 'active'
RETURNING *;

-- name: GetTrunkByID :one
SELECT t.*
FROM trunks AS t
WHERE t.id = sqlc.arg(id)
  AND t.organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetPlatformTrunkByID :one
SELECT t.*
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
WHERE t.id = sqlc.arg(id)
  AND t.organization_id IS NULL
  AND t.provisioning_mode = 'managed'
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
LIMIT 1;

-- name: ListTrunksByOrganizationID :many
SELECT t.*
FROM trunks AS t
WHERE t.organization_id = sqlc.arg(organization_id)
ORDER BY t.created_at DESC;

-- name: ListPlatformTrunks :many
SELECT t.*
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
WHERE t.organization_id IS NULL
  AND t.provisioning_mode = 'managed'
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
ORDER BY t.created_at DESC;

-- name: ListTrunksByCarrierConnectionID :many
SELECT t.*
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
JOIN carrier_providers AS cp ON cp.id = cc.provider_id
WHERE t.carrier_connection_id = sqlc.arg(carrier_connection_id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND cc.scope = 'organization'
  AND cc.organization_id = t.organization_id
  AND (
      (t.provisioning_mode = 'byoc' AND cp.slug <> 'leamout')
      OR (t.provisioning_mode = 'managed' AND cp.slug = 'leamout')
  )
ORDER BY t.created_at DESC;

-- name: UpdateTrunk :one
UPDATE trunks AS t
SET
    name = COALESCE(sqlc.narg(name), t.name),
    direction = COALESCE(sqlc.narg(direction), t.direction),
    status = COALESCE(sqlc.narg(status), t.status),
    updated_at = NOW()
WHERE t.id = sqlc.arg(id)
  AND t.organization_id = sqlc.arg(organization_id)
RETURNING t.*;

-- name: UpdatePlatformTrunk :one
UPDATE trunks AS t
SET
    name = COALESCE(sqlc.narg(name), t.name),
    direction = COALESCE(sqlc.narg(direction), t.direction),
    status = COALESCE(sqlc.narg(status), t.status),
    managed_default = COALESCE(sqlc.narg(managed_default), t.managed_default),
    updated_at = NOW()
FROM carrier_connections AS cc
WHERE t.id = sqlc.arg(id)
  AND t.organization_id IS NULL
  AND t.provisioning_mode = 'managed'
  AND cc.id = t.carrier_connection_id
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
RETURNING t.*;

-- name: DisableTrunk :one
UPDATE trunks AS t
SET
    status = 'disabled',
    updated_at = NOW()
WHERE t.id = sqlc.arg(id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND t.status = 'active'
RETURNING t.*;

-- name: EnableTrunk :exec
UPDATE trunks AS t
SET
    status = 'active',
    updated_at = NOW()
WHERE t.id = sqlc.arg(id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND t.status = 'disabled';

-- name: CreateTrunkEndpoint :one
INSERT INTO trunk_endpoints (
    organization_id,
    trunk_id,
    host,
    port,
    transport,
    direction,
    priority,
    weight,
    enabled
)
SELECT
    t.organization_id AS organization_id,
    t.id AS trunk_id,
    sqlc.arg(host) AS host,
    COALESCE(sqlc.narg(port), 5060) AS port,
    COALESCE(sqlc.narg(transport), 'udp') AS transport,
    COALESCE(sqlc.narg(direction), 'bidirectional') AS direction,
    COALESCE(sqlc.narg(priority), 10) AS priority,
    COALESCE(sqlc.narg(weight), 100) AS weight,
    COALESCE(sqlc.narg(enabled), true) AS enabled
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
JOIN carrier_providers AS cp ON cp.id = cc.provider_id
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND cc.scope = 'organization'
  AND cc.organization_id = t.organization_id
  AND (
      (t.provisioning_mode = 'byoc' AND cp.slug <> 'leamout')
      OR (t.provisioning_mode = 'managed' AND cp.slug = 'leamout')
  )
RETURNING *;

-- name: CreatePlatformTrunkEndpoint :one
INSERT INTO trunk_endpoints (
    organization_id,
    trunk_id,
    host,
    port,
    transport,
    direction,
    priority,
    weight,
    enabled
)
SELECT
    NULL::UUID AS organization_id,
    t.id AS trunk_id,
    sqlc.arg(host) AS host,
    COALESCE(sqlc.narg(port), 5060) AS port,
    COALESCE(sqlc.narg(transport), 'udp') AS transport,
    COALESCE(sqlc.narg(direction), 'bidirectional') AS direction,
    COALESCE(sqlc.narg(priority), 10) AS priority,
    COALESCE(sqlc.narg(weight), 100) AS weight,
    COALESCE(sqlc.narg(enabled), true) AS enabled
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id IS NULL
  AND t.provisioning_mode = 'managed'
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
RETURNING *;

-- name: GetTrunkEndpointByID :one
SELECT te.*
FROM trunk_endpoints AS te
JOIN trunks AS t ON t.id = te.trunk_id
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
WHERE te.id = sqlc.arg(id)
  AND te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
  AND t.organization_id = te.organization_id
  AND t.provisioning_mode = 'byoc'
  AND cc.scope = 'organization'
  AND cc.organization_id = te.organization_id
LIMIT 1;

-- name: ListTrunkEndpoints :many
SELECT te.*
FROM trunk_endpoints AS te
JOIN trunks AS t ON t.id = te.trunk_id
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
WHERE te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
  AND t.organization_id = te.organization_id
  AND t.provisioning_mode = 'byoc'
  AND cc.scope = 'organization'
  AND cc.organization_id = te.organization_id
ORDER BY te.priority ASC, te.weight DESC, te.created_at ASC;

-- name: UpdateTrunkEndpoint :one
UPDATE trunk_endpoints AS te
SET
    host = COALESCE(sqlc.narg(host), te.host),
    port = COALESCE(sqlc.narg(port), te.port),
    transport = COALESCE(sqlc.narg(transport), te.transport),
    direction = COALESCE(sqlc.narg(direction), te.direction),
    priority = COALESCE(sqlc.narg(priority), te.priority),
    weight = COALESCE(sqlc.narg(weight), te.weight),
    enabled = COALESCE(sqlc.narg(enabled), te.enabled),
    health_status = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN 'unknown'
        ELSE te.health_status
    END,
    consecutive_failures = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN 0
        ELSE te.consecutive_failures
    END,
    last_checked_at = CASE WHEN sqlc.narg(host)::TEXT IS NOT NULL OR sqlc.narg(port)::INTEGER IS NOT NULL OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL ELSE te.last_checked_at END,
    last_response_code = CASE WHEN sqlc.narg(host)::TEXT IS NOT NULL OR sqlc.narg(port)::INTEGER IS NOT NULL OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL ELSE te.last_response_code END,
    last_latency_ms = CASE WHEN sqlc.narg(host)::TEXT IS NOT NULL OR sqlc.narg(port)::INTEGER IS NOT NULL OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL ELSE te.last_latency_ms END,
    last_error = CASE WHEN sqlc.narg(host)::TEXT IS NOT NULL OR sqlc.narg(port)::INTEGER IS NOT NULL OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL ELSE te.last_error END,
    cooldown_until = CASE WHEN sqlc.narg(host)::TEXT IS NOT NULL OR sqlc.narg(port)::INTEGER IS NOT NULL OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL ELSE te.cooldown_until END,
    updated_at = NOW()
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
WHERE te.id = sqlc.arg(id)
  AND te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
  AND t.id = te.trunk_id
  AND t.organization_id = te.organization_id
  AND t.provisioning_mode = 'byoc'
  AND cc.scope = 'organization'
  AND cc.organization_id = te.organization_id
RETURNING te.*;

-- name: DeleteTrunkEndpoint :one
DELETE FROM trunk_endpoints AS te
USING trunks AS t, carrier_connections AS cc
WHERE te.id = sqlc.arg(id)
  AND te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
  AND t.id = te.trunk_id
  AND t.organization_id = te.organization_id
  AND t.provisioning_mode = 'byoc'
  AND cc.id = t.carrier_connection_id
  AND cc.scope = 'organization'
  AND cc.organization_id = te.organization_id
RETURNING te.*;

-- name: ListActiveOutboundTrunkEndpoints :many
SELECT te.*
FROM trunk_endpoints AS te
JOIN trunks AS t ON t.id = te.trunk_id
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
JOIN carrier_providers AS cp ON cp.id = cc.provider_id
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND t.status = 'active'
  AND t.direction IN ('outbound', 'bidirectional')
  AND cc.scope = 'organization'
  AND cc.organization_id = t.organization_id
  AND cc.status = 'active'
  AND te.organization_id = t.organization_id
  AND te.enabled = true
  AND te.direction IN ('outbound', 'bidirectional')
  AND (
      (t.provisioning_mode = 'byoc' AND cp.slug <> 'leamout')
      OR (t.provisioning_mode = 'managed' AND cp.slug = 'leamout')
  )
ORDER BY te.priority ASC, te.weight DESC, te.created_at ASC;

-- name: ResolveManagedOutboundRoute :many
SELECT
    cc.id AS carrier_connection_id,
    cc.max_cps,
    cc.max_concurrent_calls,
    cc.max_daily_minutes,
    t.id AS trunk_id,
    te.id AS endpoint_id,
    te.host,
    te.port,
    te.transport,
    te.priority,
    te.weight,
    te.health_status
FROM trunks AS t
JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
JOIN trunk_endpoints AS te ON te.trunk_id = t.id
WHERE t.organization_id IS NULL
  AND t.provisioning_mode = 'managed'
  AND t.managed_default = true
  AND t.status = 'active'
  AND t.direction IN ('outbound', 'bidirectional')
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
  AND cc.status = 'active'
  AND te.organization_id IS NULL
  AND te.enabled = true
  AND te.direction IN ('outbound', 'bidirectional')
ORDER BY te.priority ASC, te.weight DESC, te.created_at ASC;

-- name: ListTrunkEndpointsForHealthCheck :many
WITH due AS (
    SELECT te.id
    FROM trunk_endpoints AS te
    JOIN trunks AS t ON t.id = te.trunk_id
    JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
    WHERE te.enabled = true
      AND t.status = 'active'
      AND cc.status = 'active'
      AND te.organization_id IS NOT DISTINCT FROM t.organization_id
      AND cc.organization_id IS NOT DISTINCT FROM t.organization_id
      AND (te.cooldown_until IS NULL OR te.cooldown_until <= sqlc.arg(checked_at))
      AND (te.last_checked_at IS NULL OR te.last_checked_at <= sqlc.arg(due_before))
    ORDER BY te.last_checked_at ASC NULLS FIRST
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE OF te SKIP LOCKED
)
UPDATE trunk_endpoints AS te
SET last_checked_at = sqlc.arg(checked_at)
FROM due
WHERE te.id = due.id
RETURNING te.*;

-- name: MarkTrunkEndpointHealthy :one
UPDATE trunk_endpoints
SET health_status = 'healthy',
    consecutive_failures = 0,
    last_checked_at = sqlc.arg(checked_at),
    last_response_code = sqlc.arg(response_code),
    last_latency_ms = sqlc.arg(latency_ms),
    last_error = NULL,
    cooldown_until = NULL
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: MarkTrunkEndpointProbeFailed :one
UPDATE trunk_endpoints
SET consecutive_failures = consecutive_failures + 1,
    health_status = CASE
        WHEN consecutive_failures + 1 >= sqlc.arg(failure_threshold) THEN 'unhealthy'
        ELSE health_status
    END,
    last_checked_at = sqlc.arg(checked_at),
    last_response_code = NULL,
    last_latency_ms = sqlc.arg(latency_ms),
    last_error = sqlc.arg(last_error),
    cooldown_until = CASE
        WHEN consecutive_failures + 1 >= sqlc.arg(failure_threshold) THEN sqlc.arg(cooldown_until)::TIMESTAMPTZ
        ELSE NULL
    END
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListBackofficeTrunks :many
SELECT
    t.id::TEXT AS id,
    CAST(COALESCE(t.organization_id::TEXT, '—') AS TEXT) AS organization_id,
    COALESCE(o.name, 'Platform') AS organization_name,
    t.name,
    t.provisioning_mode,
    COALESCE(cp.name, '—') AS provider_name,
    t.direction,
    t.status,
    COUNT(te.id)::BIGINT AS endpoint_count
FROM trunks AS t
LEFT JOIN organizations AS o ON o.id = t.organization_id
LEFT JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
LEFT JOIN carrier_providers AS cp ON cp.id = cc.provider_id
LEFT JOIN trunk_endpoints AS te ON te.trunk_id = t.id
GROUP BY
    t.id,
    o.name,
    t.name,
    t.provisioning_mode,
    cp.name,
    t.direction,
    t.status,
    t.created_at
ORDER BY t.created_at DESC
LIMIT 100;

-- name: GetBackofficeTrunk :one
SELECT
    t.id::TEXT AS id,
    CAST(COALESCE(t.organization_id::TEXT, '—') AS TEXT) AS organization_id,
    COALESCE(o.name, 'Platform') AS organization_name,
    t.name,
    t.provisioning_mode,
    t.direction,
    t.status,
    t.managed_default,
    CAST(COALESCE(t.carrier_connection_id::TEXT, '—') AS TEXT) AS carrier_connection_id,
    COALESCE(cc.name, '—') AS carrier_connection_name,
    COALESCE(cp.name, '—') AS provider_name,
    COUNT(te.id)::BIGINT AS endpoint_count,
    COUNT(te.id) FILTER (WHERE te.enabled)::BIGINT AS enabled_endpoint_count,
    to_char(t.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
    to_char(t.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM trunks AS t
LEFT JOIN organizations AS o ON o.id = t.organization_id
LEFT JOIN carrier_connections AS cc ON cc.id = t.carrier_connection_id
LEFT JOIN carrier_providers AS cp ON cp.id = cc.provider_id
LEFT JOIN trunk_endpoints AS te ON te.trunk_id = t.id
WHERE t.id = sqlc.arg(id)
GROUP BY t.id, o.name, cc.name, cp.name
LIMIT 1;

-- name: ListBackofficeTrunkEndpoints :many
SELECT
    te.id::TEXT AS id,
    te.host,
    te.port,
    te.transport,
    te.direction,
    te.priority,
    te.weight,
    te.enabled,
    te.health_status,
    te.consecutive_failures,
    CAST(COALESCE(te.last_response_code::TEXT, '—') AS TEXT) AS last_response_code,
    CAST(COALESCE(te.last_latency_ms::TEXT, '—') AS TEXT) AS last_latency_ms,
    COALESCE(te.last_error, '—') AS last_error,
    CAST(COALESCE(to_char(te.last_checked_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '—') AS TEXT) AS last_checked_at
FROM trunk_endpoints AS te
WHERE te.trunk_id = sqlc.arg(trunk_id)
ORDER BY te.priority, te.host, te.port;
