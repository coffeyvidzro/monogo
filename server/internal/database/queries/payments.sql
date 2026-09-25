-- name: CreateCheckoutPayment :one
WITH payment_checkout AS MATERIALIZED (
    SELECT c.id, c.amount_minor, c.currency
    FROM checkouts AS c
    JOIN wallets AS w ON w.id = c.wallet_id
    WHERE c.id = sqlc.arg(checkout_id)::UUID
      AND w.organization_id = sqlc.arg(organization_id)::UUID
      AND c.status = 'pending' AND c.expires_at > now()
      AND c.amount_minor = sqlc.arg(amount_minor)::BIGINT
      AND c.currency = sqlc.arg(currency)::TEXT
    FOR UPDATE OF c
)
INSERT INTO payments (checkout_id, provider, attempt_key, amount_minor, currency)
SELECT c.id, sqlc.arg(provider)::TEXT, sqlc.arg(attempt_key)::TEXT,
       c.amount_minor, c.currency
FROM payment_checkout AS c
ON CONFLICT (checkout_id, provider, attempt_key) DO NOTHING
RETURNING *;

-- name: GetCheckoutPaymentByAttempt :one
SELECT p.* FROM payments AS p
JOIN checkouts AS c ON c.id = p.checkout_id
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND p.checkout_id = sqlc.arg(checkout_id)::UUID
  AND p.provider = sqlc.arg(provider)::TEXT
  AND p.attempt_key = sqlc.arg(attempt_key)::TEXT LIMIT 1;

-- name: GetCheckoutPayment :one
SELECT p.* FROM payments AS p
JOIN checkouts AS c ON c.id = p.checkout_id
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND p.id = sqlc.arg(id)::UUID LIMIT 1;

-- name: GetPaymentCheckout :one
SELECT c.* FROM checkouts AS c
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND c.id = sqlc.arg(id)::UUID
LIMIT 1;

-- name: LockCheckoutPayment :one
SELECT p.* FROM payments AS p
JOIN checkouts AS c ON c.id = p.checkout_id
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND p.id = sqlc.arg(id)::UUID
FOR UPDATE OF p;

-- name: GetProviderPaymentByReference :one
SELECT * FROM payments WHERE provider = sqlc.arg(provider)::TEXT
  AND provider_reference = sqlc.arg(provider_reference)::TEXT LIMIT 1;

-- name: RecordProviderPaymentReference :one
UPDATE payments SET provider_reference = sqlc.arg(provider_reference)::TEXT,
    status = 'pending'
WHERE id = sqlc.arg(id)::UUID AND status = 'created'
  AND provider_reference IS NULL
  AND EXISTS (
      SELECT 1 FROM checkouts AS c
      WHERE c.id = payments.checkout_id AND c.status = 'pending'
        AND c.expires_at > now()
  ) RETURNING *;

-- A verified provider result may arrive before the synchronous charge response
-- records its reference. Never overwrite a different reference.
-- name: MarkCheckoutPaymentSucceeded :one
UPDATE payments SET
    provider_reference = COALESCE(provider_reference, sqlc.arg(provider_reference)::TEXT),
    status = 'succeeded',
    verified_at = sqlc.arg(verified_at)::TIMESTAMPTZ,
    failure_code = NULL
WHERE id = sqlc.arg(id)::UUID
  AND status IN ('created', 'pending')
  AND (provider_reference IS NULL OR provider_reference = sqlc.arg(provider_reference)::TEXT)
RETURNING *;

-- name: LinkPaymentWalletTransaction :one
UPDATE payments SET wallet_transaction_id = sqlc.arg(wallet_transaction_id)::UUID
WHERE id = sqlc.arg(id)::UUID
  AND status = 'succeeded'
  AND wallet_transaction_id IS NULL
RETURNING *;

-- name: RecordIncomingPaymentEvent :one
INSERT INTO payment_events (
    payment_id, provider, provider_event_id, event_type, payload_sha256
) VALUES (
    sqlc.narg(payment_id)::UUID, sqlc.arg(provider)::TEXT,
    sqlc.arg(provider_event_id)::TEXT, sqlc.arg(event_type)::TEXT,
    sqlc.arg(payload_sha256)::TEXT
)
ON CONFLICT (provider, provider_event_id) DO NOTHING RETURNING *;

-- name: GetIncomingPaymentEventByIdentity :one
SELECT * FROM payment_events WHERE provider = sqlc.arg(provider)::TEXT
  AND provider_event_id = sqlc.arg(provider_event_id)::TEXT LIMIT 1;

-- name: MarkPaymentEventProcessed :one
UPDATE payment_events SET status = sqlc.arg(status)::TEXT,
    processed_at = now(), error_code = sqlc.narg(error_code)::TEXT
WHERE id = sqlc.arg(id)::UUID AND status = 'received'
  AND sqlc.arg(status)::TEXT IN ('processed', 'ignored', 'failed') RETURNING *;

-- name: ListPaymentsDueForRecovery :many
SELECT p.*, w.organization_id
FROM payments AS p
JOIN checkouts AS c ON c.id = p.checkout_id
JOIN wallets AS w ON w.id = c.wallet_id
WHERE p.provider_reference IS NOT NULL
  AND (
      p.status = 'pending'
      OR (p.status = 'succeeded' AND p.wallet_transaction_id IS NULL)
  )
ORDER BY p.updated_at, p.id
LIMIT sqlc.arg(row_limit);
