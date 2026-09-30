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
UPDATE checkouts AS c
SET
    status = 'processing',
    provider = sqlc.arg(provider),
    payment_method = sqlc.arg(payment_method),
    next_action = 'wait'
WHERE c.id = sqlc.arg(checkout_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND (
      c.status = 'pending'
      OR (
          c.status = 'processing'
          AND c.provider = sqlc.arg(provider)
          AND c.payment_method = sqlc.arg(payment_method)
      )
  )
  AND c.expires_at > sqlc.arg(now_at)
RETURNING c.*;

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
WITH due AS (
    SELECT id
    FROM checkouts
    WHERE status IN ('pending', 'processing')
      AND expires_at <= sqlc.arg(completed_at)
    ORDER BY expires_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(limit_count)
)
UPDATE checkouts AS c
SET
    status = 'expired',
    next_action = 'none',
    provider_message = NULL,
    completed_at = sqlc.arg(completed_at)
FROM due
WHERE c.id = due.id;
