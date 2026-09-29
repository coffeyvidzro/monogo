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
WITH updated_checkout AS (
    UPDATE checkouts
    SET
        status = 'processing',
        confirmed_at = COALESCE(confirmed_at, sqlc.arg(confirmed_at))
    WHERE id = sqlc.arg(id)
      AND organization_id = sqlc.arg(organization_id)
      AND status = 'open'
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
        sqlc.arg(provider),
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

-- name: ContinueCheckout :one
WITH target_checkout AS (
    SELECT *
    FROM checkouts
    WHERE id = sqlc.arg(id)
      AND organization_id = sqlc.arg(organization_id)
      AND status = 'failed'
    FOR UPDATE
),
next_attempt AS (
    SELECT COALESCE(MAX(p.attempt), 0) + 1 AS attempt
    FROM payments AS p
    JOIN target_checkout AS c
      ON c.id = p.checkout_id
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
        c.id,
        c.organization_id,
        sqlc.arg(provider),
        n.attempt,
        c.amount_micros,
        c.currency
    FROM target_checkout AS c
    CROSS JOIN next_attempt AS n
    RETURNING checkout_id
),
updated_checkout AS (
    UPDATE checkouts AS c
    SET
        status = 'processing',
        failure_code = NULL,
        failed_at = NULL
    FROM created_payment AS p
    WHERE c.id = p.checkout_id
    RETURNING c.*
)
SELECT *
FROM updated_checkout;

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
