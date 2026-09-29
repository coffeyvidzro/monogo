-- name: CreateCheckout :one
INSERT INTO checkouts (
    organization_id,
    purpose,
    subscription_id,
    amount_micros,
    currency,
    period_start,
    period_end
)
VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(purpose),
    sqlc.narg(subscription_id),
    sqlc.arg(amount_micros),
    sqlc.arg(currency),
    sqlc.narg(period_start),
    sqlc.narg(period_end)
)
RETURNING *;

-- name: GetCheckoutByID :one
SELECT *
FROM checkouts
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: ConfirmCheckout :one
UPDATE checkouts
SET
    status = 'processing',
    confirmed_at = COALESCE(confirmed_at, sqlc.arg(confirmed_at))
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'open'
RETURNING *;

-- name: ContinueCheckout :one
UPDATE checkouts
SET
    status = 'processing',
    failure_code = NULL,
    failed_at = NULL
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'failed'
RETURNING *;

-- name: CompleteCheckout :one
UPDATE checkouts
SET
    status = 'completed',
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'processing'
RETURNING *;

-- name: FailCheckout :one
UPDATE checkouts
SET
    status = 'failed',
    failure_code = sqlc.arg(failure_code),
    failed_at = sqlc.arg(failed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'processing'
RETURNING *;
