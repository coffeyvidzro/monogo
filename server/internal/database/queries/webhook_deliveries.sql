-- name: CreateWebhookDelivery :one
INSERT INTO webhook_deliveries (
    event_id,
    endpoint_id,
    status,
    next_attempt_at
)
SELECT
    webhook_events.id,
    webhook_endpoints.id,
    'pending',
    sqlc.arg(next_attempt_at)
FROM webhook_events
JOIN webhook_endpoints ON webhook_endpoints.id = sqlc.arg(endpoint_id)
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_events.id = sqlc.arg(event_id)
  AND webhook_endpoints.organization_id = webhook_events.organization_id
  AND webhook_endpoints.enabled = true
  AND webhook_endpoints.disabled_at IS NULL
  AND organizations.status = 'active'
ON CONFLICT (event_id, endpoint_id) DO UPDATE
SET status = EXCLUDED.status
RETURNING *;

-- name: CreateWebhookDeliveriesForEvent :execrows
INSERT INTO webhook_deliveries (event_id, endpoint_id, status, next_attempt_at)
SELECT
    sqlc.arg(event_id),
    webhook_endpoints.id,
    'pending',
    sqlc.arg(next_attempt_at)
FROM webhook_endpoints
JOIN webhook_events ON webhook_events.id = sqlc.arg(event_id)
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_endpoints.organization_id = webhook_events.organization_id
  AND webhook_endpoints.enabled = true
  AND webhook_endpoints.disabled_at IS NULL
  AND webhook_events.event_type = ANY(webhook_endpoints.subscribed_events)
  AND organizations.status = 'active'
ON CONFLICT (event_id, endpoint_id) DO NOTHING;

-- name: GetWebhookDelivery :one
SELECT webhook_deliveries.*
FROM webhook_deliveries
JOIN webhook_events ON webhook_events.id = webhook_deliveries.event_id
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_deliveries.id = sqlc.arg(id)
  AND webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
LIMIT 1;

-- name: ListWebhookDeliveriesForEvent :many
SELECT webhook_deliveries.*
FROM webhook_deliveries
JOIN webhook_events ON webhook_events.id = webhook_deliveries.event_id
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_deliveries.event_id = sqlc.arg(event_id)
  AND webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
ORDER BY webhook_deliveries.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CancelWebhookDeliveriesForEndpoint :execrows
UPDATE webhook_deliveries
SET status = 'canceled',
    last_error = 'Webhook endpoint disabled before delivery',
    locked_at = NULL,
    locked_by = NULL,
    updated_at = now()
FROM webhook_endpoints
WHERE webhook_deliveries.endpoint_id = webhook_endpoints.id
  AND webhook_deliveries.endpoint_id = sqlc.arg(endpoint_id)
  AND webhook_endpoints.organization_id = sqlc.arg(organization_id)
  AND webhook_deliveries.status IN ('pending', 'retrying');

-- name: ClaimWebhookDeliveries :many
WITH candidates AS (
    SELECT webhook_deliveries.id
    FROM webhook_deliveries
    JOIN webhook_endpoints ON webhook_endpoints.id = webhook_deliveries.endpoint_id
    JOIN webhook_events ON webhook_events.id = webhook_deliveries.event_id
    JOIN organizations ON organizations.id = webhook_events.organization_id
    WHERE webhook_deliveries.status IN ('pending', 'retrying')
      AND webhook_deliveries.next_attempt_at <= now()
      AND webhook_endpoints.enabled = true
      AND webhook_endpoints.disabled_at IS NULL
      AND organizations.status = 'active'
      AND (
          webhook_deliveries.locked_at IS NULL
          OR webhook_deliveries.locked_at < sqlc.arg(stale_before)
      )
    ORDER BY webhook_deliveries.next_attempt_at, webhook_deliveries.created_at
    FOR UPDATE OF webhook_deliveries SKIP LOCKED
    LIMIT sqlc.arg(limit_count)
)
UPDATE webhook_deliveries
SET locked_at = now(),
    locked_by = sqlc.arg(worker_id),
    attempt_count = webhook_deliveries.attempt_count + 1,
    last_attempt_at = now(),
    updated_at = now()
FROM candidates,
     webhook_events,
     webhook_endpoints
WHERE webhook_deliveries.id = candidates.id
  AND webhook_events.id = webhook_deliveries.event_id
  AND webhook_endpoints.id = webhook_deliveries.endpoint_id
RETURNING
    webhook_deliveries.id,
    webhook_deliveries.event_id,
    webhook_deliveries.endpoint_id,
    webhook_deliveries.attempt_count,
    webhook_events.organization_id,
    webhook_events.event_type,
    webhook_events.payload,
    webhook_events.occurred_at,
    webhook_endpoints.url,
    webhook_endpoints.signing_secret;

-- name: MarkWebhookDeliverySucceeded :one
WITH succeeded AS (
    UPDATE webhook_deliveries
    SET status = 'succeeded',
        response_status = sqlc.arg(response_status),
        response_body = sqlc.narg(response_body),
        last_error = NULL,
        delivered_at = now(),
        locked_at = NULL,
        locked_by = NULL,
        updated_at = now()
    WHERE webhook_deliveries.id = sqlc.arg(id)
      AND locked_by = sqlc.arg(worker_id)
    RETURNING *
), reset_endpoint AS (
    UPDATE webhook_endpoints
    SET consecutive_failures = 0,
        last_failure_at = NULL,
        disabled_reason = NULL,
        updated_at = now()
    WHERE webhook_endpoints.id = (SELECT endpoint_id FROM succeeded)
      AND enabled = true
      AND consecutive_failures > 0
    RETURNING webhook_endpoints.id
)
SELECT * FROM succeeded;

-- name: ScheduleWebhookDeliveryRetry :one
UPDATE webhook_deliveries
SET status = 'retrying',
    next_attempt_at = sqlc.arg(next_attempt_at),
    response_status = sqlc.narg(response_status),
    response_body = sqlc.narg(response_body),
    last_error = sqlc.arg(last_error),
    locked_at = NULL,
    locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND locked_by = sqlc.arg(worker_id)
RETURNING *;

-- name: MarkWebhookDeliveryFailed :one
WITH failed AS (
    UPDATE webhook_deliveries
    SET status = 'failed',
        response_status = sqlc.narg(response_status),
        response_body = sqlc.narg(response_body),
        last_error = sqlc.arg(last_error),
        locked_at = NULL,
        locked_by = NULL,
        updated_at = now()
    WHERE webhook_deliveries.id = sqlc.arg(id)
      AND locked_by = sqlc.arg(worker_id)
    RETURNING *
), update_endpoint AS (
    UPDATE webhook_endpoints
    SET consecutive_failures = consecutive_failures + 1,
        last_failure_at = now(),
        enabled = CASE
            WHEN consecutive_failures + 1 >= sqlc.arg(auto_disable_after)::integer THEN false
            ELSE enabled
        END,
        disabled_at = CASE
            WHEN consecutive_failures + 1 >= sqlc.arg(auto_disable_after)::integer THEN COALESCE(disabled_at, now())
            ELSE disabled_at
        END,
        disabled_reason = CASE
            WHEN enabled AND consecutive_failures + 1 >= sqlc.arg(auto_disable_after)::integer THEN 'failure_threshold'
            ELSE disabled_reason
        END,
        updated_at = now()
    WHERE webhook_endpoints.id = (SELECT endpoint_id FROM failed)
    RETURNING webhook_endpoints.id
)
SELECT * FROM failed;

-- name: ReleaseWebhookDeliveryClaim :execrows
UPDATE webhook_deliveries
SET locked_at = NULL,
    locked_by = NULL,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND locked_by = sqlc.arg(worker_id);

-- name: RetryWebhookDelivery :one
UPDATE webhook_deliveries
SET status = 'pending',
    next_attempt_at = now(),
    response_status = NULL,
    response_body = NULL,
    last_error = NULL,
    delivered_at = NULL,
    locked_at = NULL,
    locked_by = NULL,
    updated_at = now()
FROM webhook_events,
     webhook_endpoints,
     organizations
WHERE webhook_deliveries.id = sqlc.arg(id)
  AND webhook_events.id = webhook_deliveries.event_id
  AND webhook_endpoints.id = webhook_deliveries.endpoint_id
  AND organizations.id = webhook_events.organization_id
  AND webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
  AND webhook_endpoints.enabled = true
  AND webhook_endpoints.disabled_at IS NULL
  AND webhook_deliveries.status = 'failed'
RETURNING webhook_deliveries.*;

-- name: ReplayWebhookDelivery :one
UPDATE webhook_deliveries
SET status = 'pending',
    replay_count = webhook_deliveries.replay_count + 1,
    last_replayed_at = now(),
    next_attempt_at = now(),
    response_status = NULL,
    response_body = NULL,
    last_error = NULL,
    delivered_at = NULL,
    locked_at = NULL,
    locked_by = NULL,
    updated_at = now()
FROM webhook_events,
     webhook_endpoints,
     organizations
WHERE webhook_deliveries.id = sqlc.arg(id)
  AND webhook_events.id = webhook_deliveries.event_id
  AND webhook_endpoints.id = webhook_deliveries.endpoint_id
  AND organizations.id = webhook_events.organization_id
  AND webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
  AND webhook_endpoints.enabled = true
  AND webhook_endpoints.disabled_at IS NULL
  AND webhook_deliveries.status IN ('succeeded', 'failed', 'canceled')
RETURNING webhook_deliveries.*;

-- name: ListWebhookDeliveriesForEndpoint :many
SELECT webhook_deliveries.*
FROM webhook_deliveries
JOIN webhook_events ON webhook_events.id = webhook_deliveries.event_id
JOIN organizations ON organizations.id = webhook_events.organization_id
WHERE webhook_deliveries.endpoint_id = sqlc.arg(endpoint_id)
  AND webhook_events.organization_id = sqlc.arg(organization_id)
  AND organizations.status = 'active'
ORDER BY webhook_deliveries.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);
