-- name: CreatePaymentAttempt :one
INSERT INTO payments (
    checkout_id,
    organization_id,
    provider,
    attempt,
    amount_micros,
    currency
)
VALUES (
    sqlc.arg(checkout_id),
    sqlc.arg(organization_id),
    sqlc.arg(provider),
    sqlc.arg(attempt),
    sqlc.arg(amount_micros),
    sqlc.arg(currency)
)
RETURNING *;

-- name: GetPaymentAttemptByID :one
SELECT *
FROM payments
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetActivePaymentAttemptByCheckout :one
SELECT *
FROM payments
WHERE checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
ORDER BY attempt DESC
LIMIT 1;

-- name: GetNextPaymentAttemptNumber :one
SELECT COALESCE(MAX(attempt), 0)::bigint + 1 AS next_attempt
FROM payments
WHERE checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id);

-- name: AttachPaymentProviderReference :one
UPDATE payments
SET provider_reference = sqlc.arg(provider_reference)
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
  AND (
      provider_reference IS NULL
      OR provider_reference = sqlc.arg(provider_reference)
  )
RETURNING *;

-- name: MarkPaymentAttemptSucceeded :one
UPDATE payments
SET
    status = 'succeeded',
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
RETURNING *;

-- name: MarkPaymentAttemptFailed :one
UPDATE payments
SET
    status = 'failed',
    failure_code = sqlc.arg(failure_code),
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'pending'
RETURNING *;
