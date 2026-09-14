-- name: CreateWebhookEndpoint :one
INSERT INTO webhook_endpoints (
    organization_id,
    url,
    signing_secret,
    subscribed_events,
    enabled
)
SELECT
    sqlc.arg(organization_id) as organization_id,
    sqlc.arg(url) as url,
    sqlc.arg(signing_secret) as signing_secret,
    sqlc.arg(subscribed_events)::text[] as subscribed_events,
    COALESCE(sqlc.narg(enabled), true) as enabled
FROM organizations
WHERE organizations.id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
RETURNING *;

-- name: GetWebhookEndpointByID :one
SELECT webhook_endpoints.*
FROM webhook_endpoints
JOIN organizations ON organizations.id = webhook_endpoints.organization_id
WHERE webhook_endpoints.id = sqlc.arg(id)
  AND webhook_endpoints.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
LIMIT 1;

-- name: ListWebhookEndpointsByOrganizationID :many
SELECT *
FROM webhook_endpoints
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: UpdateWebhookEndpoint :one
UPDATE webhook_endpoints
SET
    url = COALESCE(sqlc.narg(url), url),
    signing_secret = COALESCE(sqlc.narg(signing_secret), signing_secret),
    subscribed_events = COALESCE(sqlc.narg(subscribed_events)::text[], subscribed_events),
    enabled = COALESCE(sqlc.narg(enabled), enabled),
    disabled_at = CASE WHEN sqlc.narg(enabled)::boolean = false THEN NOW() WHEN sqlc.narg(enabled)::boolean = true THEN NULL ELSE disabled_at END,
    disabled_reason = CASE WHEN sqlc.narg(enabled)::boolean = false THEN 'manual' WHEN sqlc.narg(enabled)::boolean = true THEN NULL ELSE disabled_reason END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: DisableWebhookEndpoint :exec
UPDATE webhook_endpoints
SET
    enabled = false,
    disabled_at = NOW(),
    disabled_reason = 'manual',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND enabled = true;

-- name: EnableWebhookEndpoint :exec
UPDATE webhook_endpoints
SET
    enabled = true,
    disabled_at = NULL,
    disabled_reason = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND enabled = false;

-- name: RecordWebhookEndpointFailure :one
UPDATE webhook_endpoints
SET
    consecutive_failures = consecutive_failures + 1,
    last_failure_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: ResetWebhookEndpointFailures :exec
UPDATE webhook_endpoints
SET
    consecutive_failures = 0,
    last_failure_at = NULL,
    disabled_reason = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id);

-- name: RotateWebhookEndpointSecret :one
UPDATE webhook_endpoints
SET signing_secret = sqlc.arg(signing_secret), updated_at = now()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;
