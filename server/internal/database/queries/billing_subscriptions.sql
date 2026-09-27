-- name: CreateSubscriptionPlan :one
INSERT INTO subscription_plans (
    code,
    name,
    currency,
    amount_micros
) VALUES (
    sqlc.arg(code),
    sqlc.arg(name),
    sqlc.arg(currency),
    sqlc.arg(amount_micros)
)
RETURNING *;

-- name: GetSubscriptionPlanByID :one
SELECT *
FROM subscription_plans
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: GetActiveSubscriptionPlanByCode :one
SELECT *
FROM subscription_plans
WHERE code = sqlc.arg(code)
  AND status = 'active'
LIMIT 1;

-- name: ListActiveSubscriptionPlans :many
SELECT *
FROM subscription_plans
WHERE status = 'active'
ORDER BY amount_micros ASC, code ASC;

-- name: ArchiveSubscriptionPlan :one
UPDATE subscription_plans
SET status = 'archived', updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND status = 'active'
RETURNING *;

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    organization_id,
    plan_id,
    currency,
    amount_micros,
    interval
)
SELECT
    organization.id,
    plan.id,
    plan.currency,
    plan.amount_micros,
    plan.interval
FROM organizations AS organization
JOIN subscription_plans AS plan
  ON plan.id = sqlc.arg(plan_id)
 AND plan.status = 'active'
WHERE organization.id = sqlc.arg(organization_id)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
ON CONFLICT (organization_id)
WHERE status IN ('pending', 'active', 'past_due')
DO NOTHING
RETURNING *;

-- name: GetSubscription :one
SELECT subscription.*
FROM subscriptions AS subscription
JOIN organizations AS organization
  ON organization.id = subscription.organization_id
WHERE subscription.id = sqlc.arg(id)
  AND subscription.organization_id = sqlc.arg(organization_id)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
LIMIT 1;

-- name: GetCurrentSubscription :one
SELECT subscription.*
FROM subscriptions AS subscription
JOIN organizations AS organization
  ON organization.id = subscription.organization_id
WHERE subscription.organization_id = sqlc.arg(organization_id)
  AND subscription.status IN ('pending', 'active', 'past_due')
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
ORDER BY subscription.created_at DESC
LIMIT 1;

-- name: ListSubscriptions :many
SELECT *
FROM subscriptions
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: ActivateSubscription :one
UPDATE subscriptions
SET
    status = 'active',
    current_period_start = sqlc.arg(period_start),
    current_period_end = sqlc.arg(period_end),
    started_at = COALESCE(started_at, sqlc.arg(period_start)),
    cancelled_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'past_due')
RETURNING *;

-- name: MarkSubscriptionPastDue :one
UPDATE subscriptions
SET status = 'past_due', updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: SetSubscriptionCancelAtPeriodEnd :one
UPDATE subscriptions
SET cancel_at_period_end = sqlc.arg(cancel_at_period_end), updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('active', 'past_due')
RETURNING *;

-- name: RenewSubscription :one
UPDATE subscriptions
SET
    status = 'active',
    current_period_start = sqlc.arg(period_start),
    current_period_end = sqlc.arg(period_end),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('active', 'past_due')
  AND cancel_at_period_end = false
RETURNING *;

-- name: CancelSubscription :one
UPDATE subscriptions
SET
    status = 'cancelled',
    cancel_at_period_end = false,
    cancelled_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'active', 'past_due')
RETURNING *;
