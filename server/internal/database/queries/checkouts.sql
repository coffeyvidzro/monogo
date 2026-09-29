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
WHERE id = sqlc.arg(checkout_id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: ConfirmCheckout :one
WITH updated_checkout AS (
    UPDATE checkouts AS c
    SET
        status = 'processing',
        provider = sqlc.arg(provider),
        payment_method = sqlc.arg(payment_method),
        next_action = 'wait'
    WHERE c.id = sqlc.arg(checkout_id)
      AND c.organization_id = sqlc.arg(organization_id)
      AND c.status = 'pending'
      AND c.expires_at > sqlc.arg(now_at)
    RETURNING c.*
),
created_payment AS (
    INSERT INTO payments (
        checkout_id,
        organization_id,
        provider,
        payment_method,
        attempt,
        amount_micros,
        currency
    )
    SELECT
        u.id,
        u.organization_id,
        u.provider,
        u.payment_method,
        1,
        u.amount_micros,
        u.currency
    FROM updated_checkout AS u
    RETURNING checkout_id
)
SELECT u.*
FROM updated_checkout AS u
JOIN created_payment AS p
  ON p.checkout_id = u.id;

-- name: GetCheckoutForContinuation :one
SELECT c.*
FROM checkouts AS c
WHERE c.id = sqlc.arg(checkout_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND c.status = 'processing'
  AND c.expires_at > sqlc.arg(now_at)
LIMIT 1;

-- name: UpdateCheckoutAction :one
UPDATE checkouts AS c
SET
    next_action = sqlc.arg(next_action),
    provider_message = sqlc.narg(provider_message)
WHERE c.id = sqlc.arg(checkout_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND c.status = 'processing'
RETURNING c.*;

-- name: CompleteCheckout :one
UPDATE checkouts AS c
SET
    status = 'succeeded',
    next_action = 'none',
    provider_message = NULL,
    completed_at = sqlc.arg(completed_at)
WHERE c.id = sqlc.arg(checkout_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND c.status = 'processing'
RETURNING c.*;

-- name: FailCheckout :one
UPDATE checkouts AS c
SET
    status = 'failed',
    next_action = 'none',
    provider_message = sqlc.narg(provider_message),
    failure_code = sqlc.arg(failure_code),
    completed_at = sqlc.arg(completed_at)
WHERE c.id = sqlc.arg(checkout_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND c.status = 'processing'
RETURNING c.*;

-- name: ExpireCheckout :one
UPDATE checkouts AS c
SET
    status = 'expired',
    next_action = 'none',
    completed_at = sqlc.arg(completed_at)
WHERE c.id = sqlc.arg(checkout_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND c.status IN ('pending', 'processing')
  AND c.expires_at <= sqlc.arg(completed_at)
RETURNING c.*;


-- name: ExpireDueCheckouts :execrows
UPDATE checkouts AS c
SET
    status = 'expired',
    next_action = 'none',
    provider_message = NULL,
    completed_at = sqlc.arg(completed_at)
WHERE c.status IN ('pending', 'processing')
  AND c.expires_at <= sqlc.arg(completed_at);
