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

-- name: GetActiveSubscriptionPlanByID :one
SELECT *
FROM subscription_plans
WHERE id = sqlc.arg(id)
  AND status = 'active'
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
SET
    status = 'archived',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND status = 'active'
RETURNING *;
