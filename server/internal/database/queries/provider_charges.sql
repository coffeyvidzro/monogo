-- name: CreateVoiceProviderCharge :one
INSERT INTO provider_charges (
    provider_id,
    provider_cdr_id,
    operation_id,
    provider_record_type,
    provider_record_id,
    product,
    currency,
    rate_micros,
    billable_seconds,
    amount_micros,
    incurred_at,
    raw_payload
)
SELECT
    cdr.provider_id,
    cdr.id,
    sqlc.narg(operation_id),
    'voice_cdr',
    cdr.provider_cdr_id,
    'voice',
    sqlc.arg(currency),
    sqlc.arg(rate_micros),
    sqlc.arg(billable_seconds),
    sqlc.arg(amount_micros),
    sqlc.arg(incurred_at),
    cdr.raw_payload
FROM provider_cdrs AS cdr
WHERE cdr.id = sqlc.arg(provider_cdr_id)
  AND cdr.billable_seconds = sqlc.arg(billable_seconds)
ON CONFLICT (provider_id, provider_record_type, provider_record_id) DO NOTHING
RETURNING *;

-- name: CreateProviderCharge :one
INSERT INTO provider_charges (
    provider_id,
    operation_id,
    provider_record_type,
    provider_record_id,
    product,
    currency,
    amount_micros,
    incurred_at,
    raw_payload
)
SELECT
    cp.id,
    sqlc.narg(operation_id),
    sqlc.arg(provider_record_type),
    sqlc.arg(provider_record_id),
    sqlc.arg(product),
    sqlc.arg(currency),
    sqlc.arg(amount_micros),
    sqlc.arg(incurred_at),
    sqlc.arg(raw_payload)
FROM carrier_providers AS cp
WHERE cp.id = sqlc.arg(provider_id)
  AND cp.status = 'active'
ON CONFLICT (provider_id, provider_record_type, provider_record_id) DO NOTHING
RETURNING provider_charges.*;

-- name: GetProviderChargeByID :one
SELECT * FROM provider_charges WHERE id = sqlc.arg(id) LIMIT 1;

-- name: GetProviderChargeByRecord :one
SELECT *
FROM provider_charges
WHERE provider_id = sqlc.arg(provider_id)
  AND provider_record_type = sqlc.arg(provider_record_type)
  AND provider_record_id = sqlc.arg(provider_record_id)
LIMIT 1;

-- name: GetProviderChargeByProviderCDR :one
SELECT * FROM provider_charges WHERE provider_cdr_id = sqlc.arg(provider_cdr_id) LIMIT 1;

-- name: ListProviderChargesByOperation :many
SELECT *
FROM provider_charges
WHERE operation_id = sqlc.arg(operation_id)
ORDER BY incurred_at, id;

-- name: ListUnreconciledProviderCharges :many
SELECT *
FROM provider_charges
WHERE operation_id IS NULL
ORDER BY incurred_at, id
LIMIT sqlc.arg(batch);

-- name: ListProviderChargesByCall :many
SELECT charge.*
FROM provider_charges AS charge
JOIN provider_cdrs AS cdr ON cdr.id = charge.provider_cdr_id
WHERE cdr.call_id = sqlc.arg(call_id)
ORDER BY charge.incurred_at, charge.id;

-- name: ReconcileProviderChargeOperation :one
UPDATE provider_charges
SET operation_id = sqlc.arg(operation_id)
WHERE id = sqlc.arg(id)
  AND operation_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM wallet_ledger_entries AS ledger
      WHERE ledger.operation_id = sqlc.arg(operation_id)
        AND ledger.direction = 'debit'
  )
RETURNING *;
