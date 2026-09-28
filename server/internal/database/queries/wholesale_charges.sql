-- name: CreateWholesaleCharge :one
INSERT INTO wholesale_charges (
    provider_cdr_id,
    currency,
    rate_micros,
    billable_seconds,
    amount_micros,
    rated_at
)
SELECT
    cdr.id,
    sqlc.arg(currency),
    sqlc.arg(rate_micros),
    sqlc.arg(billable_seconds),
    sqlc.arg(amount_micros),
    sqlc.arg(rated_at)
FROM provider_cdrs AS cdr
WHERE cdr.id = sqlc.arg(provider_cdr_id)
  AND cdr.billable_seconds = sqlc.arg(billable_seconds)
ON CONFLICT (provider_cdr_id) DO NOTHING
RETURNING *;

-- name: GetWholesaleChargeByID :one
SELECT *
FROM wholesale_charges
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: GetWholesaleChargeByProviderCDR :one
SELECT *
FROM wholesale_charges
WHERE provider_cdr_id = sqlc.arg(provider_cdr_id)
LIMIT 1;

-- name: ListWholesaleChargesByCall :many
SELECT wc.*
FROM wholesale_charges AS wc
JOIN provider_cdrs AS cdr
  ON cdr.id = wc.provider_cdr_id
WHERE cdr.call_id = sqlc.arg(call_id)
ORDER BY wc.rated_at ASC, wc.id ASC;
