-- name: CreatePaymentAttempt :one
INSERT INTO payments (
    checkout_id,
    organization_id,
    provider,
    payment_method,
    attempt,
    amount_micros,
    currency
)
VALUES (
    sqlc.arg(checkout_id),
    sqlc.arg(organization_id),
    sqlc.arg(provider),
    sqlc.arg(payment_method),
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

-- name: GetPaymentAttemptByProviderPaymentID :one
SELECT *
FROM payments
WHERE provider = sqlc.arg(provider)
  AND provider_payment_id = sqlc.arg(provider_payment_id)
LIMIT 1;

-- name: GetActivePaymentAttemptByCheckout :one
SELECT *
FROM payments
WHERE checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'processing')
ORDER BY attempt DESC
LIMIT 1;

-- name: GetNextPaymentAttemptNumber :one
SELECT COALESCE(MAX(attempt), 0)::bigint + 1 AS next_attempt
FROM payments
WHERE checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id);

-- name: AttachProviderPaymentID :one
UPDATE payments
SET
    provider_payment_id = sqlc.arg(provider_payment_id),
    status = 'processing'
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'processing')
  AND (
      provider_payment_id IS NULL
      OR provider_payment_id = sqlc.arg(provider_payment_id)
  )
RETURNING *;

-- name: MarkPaymentAttemptSucceeded :one
UPDATE payments
SET
    status = 'succeeded',
    failure_code = NULL,
    paid_at = COALESCE(paid_at, sqlc.arg(paid_at))
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'processing', 'succeeded')
RETURNING *;

-- name: MarkPaymentAttemptFailed :one
UPDATE payments
SET
    status = 'failed',
    failure_code = sqlc.arg(failure_code)
WHERE id = sqlc.arg(id)
  AND checkout_id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'processing')
RETURNING *;
