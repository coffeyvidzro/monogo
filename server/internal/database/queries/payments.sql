-- name: CreatePayment :one
INSERT INTO payments (
    organization_id,
    purpose,
    provider,
    subscription_id,
    amount_micros,
    currency,
    period_start,
    period_end
)
VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(purpose),
    sqlc.arg(provider),
    sqlc.narg(subscription_id),
    sqlc.arg(amount_micros),
    sqlc.arg(currency),
    sqlc.narg(period_start),
    sqlc.narg(period_end)
)
RETURNING *;

-- name: GetPaymentByID :one
SELECT *
FROM payments
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: AttachPaymentProviderReference :one
UPDATE payments
SET provider_reference = sqlc.arg(provider_reference)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
  AND (
      provider_reference IS NULL
      OR provider_reference = sqlc.arg(provider_reference)
  )
RETURNING *;

-- name: ClaimPaymentProviderEvent :one
UPDATE payments
SET provider_event_id = sqlc.arg(provider_event_id)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
  AND (
      provider_event_id IS NULL
      OR provider_event_id = sqlc.arg(provider_event_id)
  )
RETURNING *;

-- name: MarkPaymentSucceeded :one
UPDATE payments
SET
    status = 'succeeded',
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
  AND provider_event_id = sqlc.arg(provider_event_id)
RETURNING *;

-- name: MarkPaymentFailed :one
UPDATE payments
SET
    status = 'failed',
    failure_code = sqlc.arg(failure_code),
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
  AND provider_event_id = sqlc.arg(provider_event_id)
RETURNING *;
