-- name: CreateTrunk :one
INSERT INTO trunks (
    id,
    organization_id,
    name,
    direction,
    status,
    outbound_auth_method,
    auth_username,
    auth_realm,
    auth_secret_ciphertext,
    inbound_enabled,
    inbound_auth_method,
    inbound_username,
    inbound_realm,
    inbound_secret_ciphertext,
    max_cps,
    max_concurrent_calls,
    codecs,
    supports_video,
    supports_fax
)
SELECT
    sqlc.arg(id) AS id,
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(name) AS name,
    COALESCE(sqlc.narg(direction), 'bidirectional') AS direction,
    COALESCE(sqlc.narg(status), 'active') AS status,
    COALESCE(sqlc.narg(outbound_auth_method), 'none') AS outbound_auth_method,
    sqlc.narg(auth_username) AS auth_username,
    sqlc.narg(auth_realm) AS auth_realm,
    sqlc.narg(auth_secret_ciphertext) AS auth_secret_ciphertext,
    COALESCE(sqlc.narg(inbound_enabled), false) AS inbound_enabled,
    COALESCE(sqlc.narg(inbound_auth_method), 'ip') AS inbound_auth_method,
    sqlc.narg(inbound_username) AS inbound_username,
    sqlc.narg(inbound_realm) AS inbound_realm,
    sqlc.narg(inbound_secret_ciphertext) AS inbound_secret_ciphertext,
    COALESCE(sqlc.narg(max_cps), 10) AS max_cps,
    COALESCE(sqlc.narg(max_concurrent_calls), 100) AS max_concurrent_calls,
    COALESCE(sqlc.narg(codecs), ARRAY['PCMU','PCMA']::TEXT[]) AS codecs,
    COALESCE(sqlc.narg(supports_video), false) AS supports_video,
    COALESCE(sqlc.narg(supports_fax), false) AS supports_fax
FROM organizations AS o
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: InsertTrunkDigestCredential :exec
INSERT INTO trunk_digest_credentials (
    trunk_id,
    organization_id,
    direction,
    username,
    realm,
    ha1_md5
)
SELECT
    t.id,
    t.organization_id,
    sqlc.arg(direction)::TEXT,
    sqlc.arg(username)::TEXT,
    sqlc.arg(realm)::TEXT,
    sqlc.arg(ha1_md5)::TEXT
FROM trunks AS t
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id);

-- name: GetTrunkByID :one
SELECT *
FROM trunks
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: ListTrunksByOrganizationID :many
SELECT *
FROM trunks
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: UpdateTrunk :one
UPDATE trunks
SET
    name = COALESCE(sqlc.narg(name), name),
    direction = COALESCE(sqlc.narg(direction), direction),
    status = COALESCE(sqlc.narg(status), status),
    inbound_enabled = COALESCE(sqlc.narg(inbound_enabled), inbound_enabled),
    max_cps = COALESCE(sqlc.narg(max_cps), max_cps),
    max_concurrent_calls = COALESCE(sqlc.narg(max_concurrent_calls), max_concurrent_calls),
    codecs = COALESCE(sqlc.narg(codecs), codecs),
    supports_video = COALESCE(sqlc.narg(supports_video), supports_video),
    supports_fax = COALESCE(sqlc.narg(supports_fax), supports_fax),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: DisableTrunk :one
UPDATE trunks
SET status = 'disabled',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: SetTrunkOutboundDigestAuth :exec
WITH updated AS (
    UPDATE trunks AS t
    SET outbound_auth_method = 'digest',
        auth_username = sqlc.narg(auth_username),
        auth_realm = sqlc.narg(auth_realm),
        auth_secret_ciphertext = sqlc.narg(auth_secret_ciphertext),
        updated_at = NOW()
    WHERE t.id = sqlc.arg(id)
      AND t.organization_id = sqlc.arg(organization_id)
    RETURNING t.id, t.organization_id
)
INSERT INTO trunk_digest_credentials (
    trunk_id,
    organization_id,
    direction,
    username,
    realm,
    ha1_md5
)
SELECT
    updated.id,
    updated.organization_id,
    'outbound',
    sqlc.narg(auth_username),
    sqlc.narg(auth_realm),
    sqlc.narg(auth_ha1_md5)
FROM updated
ON CONFLICT (trunk_id, direction)
DO UPDATE SET
    username = EXCLUDED.username,
    realm = EXCLUDED.realm,
    ha1_md5 = EXCLUDED.ha1_md5,
    updated_at = NOW();

-- name: ClearTrunkOutboundAuth :exec
WITH updated AS (
    UPDATE trunks AS t
    SET outbound_auth_method = 'none',
        auth_username = NULL,
        auth_realm = NULL,
        auth_secret_ciphertext = NULL,
        updated_at = NOW()
    WHERE t.id = sqlc.arg(id)
      AND t.organization_id = sqlc.arg(organization_id)
    RETURNING t.id
)
DELETE FROM trunk_digest_credentials AS d
USING updated
WHERE d.trunk_id = updated.id
  AND d.direction = 'outbound';

-- name: SetTrunkInboundDigestAuth :exec
WITH updated AS (
    UPDATE trunks AS t
    SET inbound_auth_method = 'digest',
        inbound_username = sqlc.narg(inbound_username),
        inbound_realm = sqlc.narg(inbound_realm),
        inbound_secret_ciphertext = sqlc.narg(inbound_secret_ciphertext),
        updated_at = NOW()
    WHERE t.id = sqlc.arg(id)
      AND t.organization_id = sqlc.arg(organization_id)
    RETURNING t.id, t.organization_id
)
INSERT INTO trunk_digest_credentials (
    trunk_id,
    organization_id,
    direction,
    username,
    realm,
    ha1_md5
)
SELECT
    updated.id,
    updated.organization_id,
    'inbound',
    sqlc.narg(inbound_username),
    sqlc.narg(inbound_realm),
    sqlc.narg(inbound_ha1_md5)
FROM updated
ON CONFLICT (trunk_id, direction)
DO UPDATE SET
    username = EXCLUDED.username,
    realm = EXCLUDED.realm,
    ha1_md5 = EXCLUDED.ha1_md5,
    updated_at = NOW();

-- name: SetTrunkInboundIPAuth :exec
WITH updated AS (
    UPDATE trunks AS t
    SET inbound_auth_method = 'ip',
        inbound_username = NULL,
        inbound_realm = NULL,
        inbound_secret_ciphertext = NULL,
        updated_at = NOW()
    WHERE t.id = sqlc.arg(id)
      AND t.organization_id = sqlc.arg(organization_id)
    RETURNING t.id
)
DELETE FROM trunk_digest_credentials AS d
USING updated
WHERE d.trunk_id = updated.id
  AND d.direction = 'inbound';

-- name: SetTrunkInboundNoAuth :exec
WITH updated AS (
    UPDATE trunks AS t
    SET inbound_auth_method = 'none',
        inbound_username = NULL,
        inbound_realm = NULL,
        inbound_secret_ciphertext = NULL,
        updated_at = NOW()
    WHERE t.id = sqlc.arg(id)
      AND t.organization_id = sqlc.arg(organization_id)
    RETURNING t.id
)
DELETE FROM trunk_digest_credentials AS d
USING updated
WHERE d.trunk_id = updated.id
  AND d.direction = 'inbound';

-- name: CreateTrunkSourceIP :one
INSERT INTO trunk_source_ips (
    organization_id,
    trunk_id,
    cidr
)
SELECT
    t.organization_id AS organization_id,
    t.id AS trunk_id,
    sqlc.arg(cidr) AS cidr
FROM trunks AS t
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: ListTrunkSourceIPs :many
SELECT *
FROM trunk_source_ips
WHERE trunk_id = sqlc.arg(trunk_id)
  AND organization_id = sqlc.arg(organization_id)
ORDER BY created_at ASC;

-- name: DeleteTrunkSourceIP :exec
DELETE FROM trunk_source_ips
WHERE id = sqlc.arg(id)
  AND trunk_id = sqlc.arg(trunk_id)
  AND organization_id = sqlc.arg(organization_id);

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
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: GetTrunkEndpointByID :one
SELECT te.*
FROM trunk_endpoints AS te
JOIN trunks AS t
  ON t.id = te.trunk_id
 AND t.organization_id = te.organization_id
WHERE te.id = sqlc.arg(id)
  AND te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: ListTrunkEndpoints :many
SELECT te.*
FROM trunk_endpoints AS te
JOIN trunks AS t
  ON t.id = te.trunk_id
 AND t.organization_id = te.organization_id
WHERE te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
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
    last_checked_at = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL
        ELSE te.last_checked_at
    END,
    last_response_code = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL
        ELSE te.last_response_code
    END,
    last_latency_ms = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL
        ELSE te.last_latency_ms
    END,
    last_error = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL
        ELSE te.last_error
    END,
    cooldown_until = CASE
        WHEN sqlc.narg(host)::TEXT IS NOT NULL
          OR sqlc.narg(port)::INTEGER IS NOT NULL
          OR sqlc.narg(transport)::TEXT IS NOT NULL THEN NULL
        ELSE te.cooldown_until
    END,
    updated_at = NOW()
FROM trunks AS t
WHERE te.id = sqlc.arg(id)
  AND te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
  AND t.id = te.trunk_id
  AND t.organization_id = te.organization_id
RETURNING te.*;

-- name: DeleteTrunkEndpoint :one
DELETE FROM trunk_endpoints AS te
USING trunks AS t
WHERE te.id = sqlc.arg(id)
  AND te.trunk_id = sqlc.arg(trunk_id)
  AND te.organization_id = sqlc.arg(organization_id)
  AND t.id = te.trunk_id
  AND t.organization_id = te.organization_id
RETURNING te.*;

-- name: ListActiveOutboundTrunkEndpoints :many
SELECT te.*
FROM trunk_endpoints AS te
JOIN trunks AS t
  ON t.id = te.trunk_id
 AND t.organization_id = te.organization_id
WHERE t.id = sqlc.arg(trunk_id)
  AND t.organization_id = sqlc.arg(organization_id)
  AND t.status = 'active'
  AND t.direction IN ('outbound', 'bidirectional')
  AND te.enabled = true
  AND te.direction IN ('outbound', 'bidirectional')
ORDER BY te.priority ASC, te.weight DESC, te.created_at ASC;

-- name: ListTrunkEndpointsForHealthCheck :many
WITH due AS (
    SELECT te.id
    FROM trunk_endpoints AS te
    JOIN trunks AS t
      ON t.id = te.trunk_id
     AND t.organization_id = te.organization_id
    WHERE te.enabled = true
      AND t.status = 'active'
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
SET
    health_status = 'healthy',
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
SET
    consecutive_failures = consecutive_failures + 1,
    health_status = CASE
        WHEN consecutive_failures + 1 >= sqlc.arg(failure_threshold) THEN 'unhealthy'
        ELSE health_status
    END,
    last_checked_at = sqlc.arg(checked_at),
    last_response_code = NULL,
    last_latency_ms = sqlc.arg(latency_ms),
    last_error = sqlc.arg(last_error),
    cooldown_until = CASE
        WHEN consecutive_failures + 1 >= sqlc.arg(failure_threshold)
            THEN sqlc.arg(cooldown_until)::TIMESTAMPTZ
        ELSE NULL
    END
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListBackofficeTrunks :many
SELECT
    t.id::TEXT AS id,
    t.organization_id::TEXT AS organization_id,
    o.name AS organization_name,
    t.name,
    t.direction,
    t.status,
    COUNT(te.id)::BIGINT AS endpoint_count
FROM trunks AS t
JOIN organizations AS o ON o.id = t.organization_id
LEFT JOIN trunk_endpoints AS te ON te.trunk_id = t.id
GROUP BY
    t.id,
    o.name,
    t.name,
    t.direction,
    t.status,
    t.created_at
ORDER BY t.created_at DESC
LIMIT 100;

-- name: GetBackofficeTrunk :one
SELECT
    t.id::TEXT AS id,
    t.organization_id::TEXT AS organization_id,
    o.name AS organization_name,
    t.name,
    t.direction,
    t.status,
    t.outbound_auth_method,
    t.inbound_enabled,
    t.inbound_auth_method,
    t.max_cps,
    t.max_concurrent_calls,
    COUNT(te.id)::BIGINT AS endpoint_count,
    COUNT(te.id) FILTER (WHERE te.enabled)::BIGINT AS enabled_endpoint_count,
    to_char(t.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
    to_char(t.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM trunks AS t
JOIN organizations AS o ON o.id = t.organization_id
LEFT JOIN trunk_endpoints AS te ON te.trunk_id = t.id
WHERE t.id = sqlc.arg(id)
GROUP BY t.id, o.name
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
    CAST(
        COALESCE(
            to_char(te.last_checked_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'),
            '—'
        ) AS TEXT
    ) AS last_checked_at
FROM trunk_endpoints AS te
WHERE te.trunk_id = sqlc.arg(trunk_id)
ORDER BY te.priority, te.host, te.port;
