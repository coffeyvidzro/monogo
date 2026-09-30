-- name: CreateSubscription :one
INSERT INTO subscriptions (
    organization_id,
    plan_id,
    status,
    currency,
    amount_micros,
    interval
)
SELECT
    o.id,
    p.id,
    'pending',
    p.currency,
    p.amount_micros,
    p.interval
FROM organizations AS o
JOIN subscription_plans AS p
  ON p.id = sqlc.arg(plan_id)
 AND p.status = 'active'
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT s.*
FROM subscriptions AS s
JOIN organizations AS o
  ON o.id = s.organization_id
WHERE s.id = sqlc.arg(id)
  AND s.organization_id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListSubscriptionsByOrganization :many
SELECT s.*
FROM subscriptions AS s
JOIN organizations AS o
  ON o.id = s.organization_id
WHERE s.organization_id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ORDER BY s.created_at DESC, s.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: GetCurrentSubscriptionByOrganization :one
SELECT s.*
FROM subscriptions AS s
JOIN organizations AS o
  ON o.id = s.organization_id
WHERE s.organization_id = sqlc.arg(organization_id)
  AND s.status IN ('pending', 'active', 'past_due')
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ORDER BY s.created_at DESC
LIMIT 1;

-- name: ActivateSubscription :one
UPDATE subscriptions AS s
SET
    status = 'active',
    current_period_start = CASE
        WHEN s.status = 'active' THEN s.current_period_start
        ELSE sqlc.arg(current_period_start)
    END,
    current_period_end = CASE
        WHEN s.status = 'active' THEN s.current_period_end
        ELSE sqlc.arg(current_period_end)
    END,
    started_at = COALESCE(s.started_at, sqlc.arg(current_period_start)),
    updated_at = NOW()
WHERE s.id = sqlc.arg(id)
  AND s.organization_id = sqlc.arg(organization_id)
  AND s.status IN ('pending', 'past_due', 'active')
  AND EXISTS (
      SELECT 1
      FROM organizations AS o
      WHERE o.id = s.organization_id
        AND o.status = 'active'
        AND o.deleted_at IS NULL
  )
RETURNING s.*;

-- name: MarkSubscriptionPastDue :one
UPDATE subscriptions
SET
    status = 'past_due',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: UpdateSubscriptionCancelAtPeriodEnd :one
UPDATE subscriptions
SET
    cancel_at_period_end = sqlc.arg(cancel_at_period_end),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('active', 'past_due')
RETURNING *;

-- name: CancelSubscription :one
UPDATE subscriptions
SET
    status = 'cancelled',
    cancel_at_period_end = false,
    cancelled_at = sqlc.arg(cancelled_at),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'active', 'past_due')
RETURNING *;
