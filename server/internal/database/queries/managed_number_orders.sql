-- name: CreateManagedNumberOrder :one
INSERT INTO managed_number_orders (
    id, organization_id, provider_id, idempotency_key, request_hash,
    number, country_code, available_did_id, sku_id
)
SELECT sqlc.arg(id), o.id, cp.id, sqlc.arg(idempotency_key),
       sqlc.arg(request_hash), sqlc.arg(number), sqlc.arg(country_code),
       sqlc.arg(available_did_id), sqlc.arg(sku_id)
FROM organizations AS o
JOIN carrier_providers AS cp
  ON cp.slug = 'didww' AND cp.adapter = 'didww' AND cp.status = 'active'
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (organization_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetManagedNumberOrderByKey :one
SELECT *
FROM managed_number_orders
WHERE organization_id = sqlc.arg(organization_id)
  AND idempotency_key = sqlc.arg(idempotency_key)
LIMIT 1;

-- name: GetManagedNumberOrder :one
SELECT *
FROM managed_number_orders
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: GetManagedNumberOrderInternal :one
SELECT *
FROM managed_number_orders
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: LockManagedNumberOrder :one
SELECT *
FROM managed_number_orders
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: ListManagedNumberOrdersForReconciliation :many
SELECT *
FROM managed_number_orders
WHERE status IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring')
  AND reconcile_after <= now()
ORDER BY reconcile_after, created_at
LIMIT sqlc.arg(batch_size);

-- name: ClaimManagedNumberOrderSubmission :one
UPDATE managed_number_orders
SET status = 'submitting', submitted_at = now(),
    error_code = NULL, error_message = NULL
WHERE id = sqlc.arg(id) AND status = 'ready'
RETURNING *;

-- name: MarkManagedNumberOrderOutcomeUnknown :one
UPDATE managed_number_orders
SET status = 'outcome_unknown', error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message),
    reconcile_after = sqlc.arg(reconcile_after)
WHERE id = sqlc.arg(id) AND status = 'submitting'
RETURNING *;

-- name: RecordManagedNumberProviderOrder :one
UPDATE managed_number_orders
SET provider_order_id = sqlc.arg(provider_order_id), status = 'provider_pending',
    error_code = NULL, error_message = NULL,
    reconcile_after = sqlc.arg(reconcile_after)
WHERE id = sqlc.arg(id)
  AND status IN ('submitting', 'outcome_unknown', 'provider_pending')
  AND (provider_order_id IS NULL OR provider_order_id = sqlc.arg(provider_order_id))
RETURNING *;

-- name: RecordManagedNumberOwnedDID :one
UPDATE managed_number_orders
SET provider_did_id = sqlc.arg(provider_did_id),
    ownership_verified_at = sqlc.arg(verified_at), status = 'configuring',
    error_code = NULL, error_message = NULL
WHERE id = sqlc.arg(id) AND status IN ('provider_pending', 'configuring')
  AND (provider_did_id IS NULL OR provider_did_id = sqlc.arg(provider_did_id))
RETURNING *;

-- name: ScheduleManagedNumberReconciliation :one
UPDATE managed_number_orders
SET reconcile_after = sqlc.arg(reconcile_after),
    reconcile_attempts = reconcile_attempts + 1,
    error_code = sqlc.arg(error_code), error_message = NULL
WHERE id = sqlc.arg(id)
  AND status IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring')
RETURNING *;

-- name: MarkManagedNumberOrderManualReview :one
UPDATE managed_number_orders
SET status = 'manual_review', error_code = sqlc.arg(error_code),
    error_message = 'provider identity requires manual review'
WHERE id = sqlc.arg(id) AND status <> 'completed'
RETURNING *;

-- name: CompleteManagedNumberOrder :one
UPDATE managed_number_orders
SET phone_number_id = sqlc.arg(phone_number_id),
    inbound_trunk_id = sqlc.arg(inbound_trunk_id),
    routing_verified_at = sqlc.arg(verified_at),
    activated_at = sqlc.arg(verified_at), status = 'completed',
    error_code = NULL, error_message = NULL
WHERE id = sqlc.arg(id) AND status = 'configuring'
RETURNING *;
