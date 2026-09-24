-- These are internal purchase primitives, not an HTTP purchase API.
-- A trusted server-side quote/selection must be validated before the caller
-- creates an intent. A zero-row insert can mean a replay, conflicting intent,
-- unavailable tenant/provider, or an expired quote; inspect the existing key
-- before deciding whether to return a replay or a conflict.
--
-- name: CreateManagedNumberPurchaseIntent :one
INSERT INTO managed_number_orders (
    id,
    organization_id,
    provider_id,
    idempotency_key,
    request_hash,
    number,
    country_code,
    available_did_id,
    sku_id,
    quote_id,
    purchase_amount_minor,
    recurring_amount_minor,
    currency,
    quote_expires_at
)
SELECT
    sqlc.arg(id)::UUID,
    o.id,
    cp.id,
    sqlc.arg(idempotency_key),
    sqlc.arg(request_hash),
    sqlc.arg(number),
    sqlc.arg(country_code),
    sqlc.arg(available_did_id),
    sqlc.arg(sku_id),
    sqlc.arg(quote_id)::UUID,
    sqlc.arg(purchase_amount_minor)::BIGINT,
    sqlc.arg(recurring_amount_minor)::BIGINT,
    sqlc.arg(currency),
    sqlc.arg(quote_expires_at)::TIMESTAMPTZ
FROM organizations AS o
JOIN carrier_providers AS cp ON cp.slug = 'didww' AND cp.adapter = 'didww' AND cp.status = 'active'
WHERE o.id = sqlc.arg(organization_id)::UUID
  AND o.status = 'active'
  AND o.deleted_at IS NULL
  AND sqlc.arg(quote_expires_at)::TIMESTAMPTZ > now()
ON CONFLICT DO NOTHING
RETURNING *;

-- name: GetManagedNumberPurchaseIntentByKey :one
SELECT *
FROM managed_number_orders
WHERE organization_id = sqlc.arg(organization_id)
  AND idempotency_key = sqlc.arg(idempotency_key)
LIMIT 1;

-- name: GetManagedNumberPurchaseIntentByID :one
SELECT *
FROM managed_number_orders
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: LockManagedNumberPurchaseIntent :one
SELECT *
FROM managed_number_orders
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
FOR UPDATE;

-- A wallet integration must actually reserve the accepted amount before
-- invoking this transition. This UUID is a reference, not proof of funds.
--
-- name: MarkManagedNumberPurchaseFunded :one
UPDATE managed_number_orders
SET wallet_reservation_id = sqlc.arg(wallet_reservation_id),
    status = 'ready'
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'pending_funding'
  AND wallet_reservation_id IS NULL
  AND quote_expires_at > now()
RETURNING *;

-- Claim exactly once before any network call. An interrupted submission must
-- be reconciled; it must never be blindly moved back to 'ready'.
--
-- name: ClaimManagedNumberPurchaseSubmission :one
UPDATE managed_number_orders
SET status = 'submitting',
    submitted_at = now()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'ready'
  AND wallet_reservation_id IS NOT NULL
  AND quote_expires_at > now()
RETURNING *;

-- name: MarkManagedNumberPurchaseOutcomeUnknown :one
UPDATE managed_number_orders
SET status = 'outcome_unknown'
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'submitting'
RETURNING *;

-- Only after reconciliation has matched the upstream order to this internal
-- intent, including the original external reference, should it be persisted.
--
-- name: RecordManagedNumberProviderOrder :one
UPDATE managed_number_orders
SET provider_order_id = sqlc.arg(provider_order_id),
    status = 'provider_pending'
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status IN ('submitting', 'outcome_unknown', 'provider_pending')
  AND (provider_order_id IS NULL OR provider_order_id = sqlc.arg(provider_order_id))
RETURNING *;

-- Only after the provider confirms ownership of the exact selected number
-- and DID may the worker advance to inbound-trunk provisioning.
--
-- name: RecordManagedNumberAcquiredDID :one
UPDATE managed_number_orders
SET provider_did_id = sqlc.arg(provider_did_id),
    status = 'configuring'
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'provider_pending'
  AND provider_order_id IS NOT NULL
  AND provider_did_id IS NULL
RETURNING *;

-- Failure before submitting to the provider is safe to record without
-- upstream reconciliation; after submission it is not.
--
-- name: FailUnsubmittedManagedNumberPurchase :one
UPDATE managed_number_orders
SET status = 'failed',
    error_code = sqlc.arg(error_code),
    error_message = sqlc.arg(error_message)
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status IN ('pending_funding', 'ready')
  AND submitted_at IS NULL
RETURNING *;
