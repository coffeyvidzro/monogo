-- name: CreateProviderCost :one
INSERT INTO provider_costs (
    provider_id,
    operation_id,
    provider_record_id,
    product,
    currency,
    amount_micros,
    incurred_at,
    raw_payload
)
SELECT
    cp.id,
    ledger.operation_id,
    sqlc.arg(provider_record_id),
    sqlc.arg(product),
    sqlc.arg(currency),
    sqlc.arg(amount_micros),
    sqlc.arg(incurred_at),
    sqlc.arg(raw_payload)
FROM carrier_providers AS cp
JOIN wallet_ledger_entries AS ledger
  ON ledger.operation_id = sqlc.arg(operation_id)
 AND ledger.direction = 'debit'
WHERE cp.id = sqlc.arg(provider_id)
  AND cp.status = 'active'
ON CONFLICT (provider_id, provider_record_id) DO NOTHING
RETURNING provider_costs.*;

-- name: GetProviderCostByProviderRecord :one
SELECT *
FROM provider_costs
WHERE provider_id = sqlc.arg(provider_id)
  AND provider_record_id = sqlc.arg(provider_record_id)
LIMIT 1;

-- name: ListProviderCostsByOperation :many
SELECT *
FROM provider_costs
WHERE operation_id = sqlc.arg(operation_id)
ORDER BY incurred_at, id;
