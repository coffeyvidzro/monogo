-- name: CreateCheckout :one
INSERT INTO checkouts (
    organization_id,
    purpose,
    subscription_id,
    reference,
    amount_micros,
    currency,
    expires_at
)
VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(purpose),
    sqlc.narg(subscription_id),
    sqlc.arg(reference),
    sqlc.arg(amount_micros),
    sqlc.arg(currency),
    sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetCheckoutByID :one
SELECT *
FROM checkouts
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: ConfirmCheckout :one
WITH updated_checkout AS (
    UPDATE checkouts
    SET
        status = 'processing',
        provider = sqlc.arg(provider),
        payment_method = sqlc.arg(payment_method),
        next_action = 'wait'
    WHERE id = sqlc.arg(id)
      AND organization_id = sqlc.arg(organization_id)
      AND status = 'pending'
      AND expires_at > sqlc.arg(now_at)
    RETURNING *
),
created_payment AS (
    INSERT INTO payments (
        checkout_id,
        organization_id,
        provider,
        attempt,
        amount_micros,
        currency
    )
    SELECT
        id,
        organization_id,
        provider,
        1,
        amount_micros,
        currency
    FROM updated_checkout
    RETURNING checkout_id
)
SELECT updated_checkout.*
FROM updated_checkout
JOIN created_payment
  ON created_payment.checkout_id = updated_checkout.id;

-- name: GetCheckoutForContinuation :one
SELECT *
FROM checkouts
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'processing'
  AND expires_at > sqlc.arg(now_at)
LIMIT 1;

-- name: UpdateCheckoutAction :one
UPDATE checkouts
SET
    next_action = sqlc.arg(next_action),
    provider_message = sqlc.narg(provider_message)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'processing'
RETURNING *;

-- name: CompleteCheckout :one
UPDATE checkouts
SET
    status = 'succeeded',
    next_action = 'none',
    provider_message = NULL,
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'processing'
RETURNING *;

-- name: FailCheckout :one
UPDATE checkouts
SET
    status = 'failed',
    next_action = 'none',
    provider_message = sqlc.narg(provider_message),
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'processing'
RETURNING *;

-- name: ExpireCheckout :one
UPDATE checkouts
SET
    status = 'expired',
    next_action = 'none',
    completed_at = sqlc.arg(completed_at)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('pending', 'processing')
  AND expires_at <= sqlc.arg(completed_at)
RETURNING *;
