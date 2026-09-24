-- name: CreateWalletCheckout :one
INSERT INTO checkouts (wallet_id, amount_minor, currency, idempotency_key, request_hash, expires_at)
SELECT w.id, sqlc.arg(amount_minor)::BIGINT, w.currency,
       sqlc.arg(idempotency_key)::TEXT, sqlc.arg(request_hash)::TEXT,
       sqlc.arg(expires_at)::TIMESTAMPTZ
FROM wallets AS w
JOIN organizations AS o ON o.id = w.organization_id
WHERE w.id = sqlc.arg(wallet_id)::UUID
  AND w.organization_id = sqlc.arg(organization_id)::UUID
  AND w.currency = sqlc.arg(currency)::TEXT
  AND o.status = 'active' AND o.deleted_at IS NULL
  AND sqlc.arg(expires_at)::TIMESTAMPTZ > now()
ON CONFLICT (wallet_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetWalletCheckoutByKey :one
SELECT c.* FROM checkouts AS c
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND c.wallet_id = sqlc.arg(wallet_id)::UUID
  AND c.idempotency_key = sqlc.arg(idempotency_key)::TEXT LIMIT 1;

-- name: GetWalletCheckout :one
SELECT c.* FROM checkouts AS c
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND c.id = sqlc.arg(id)::UUID LIMIT 1;

-- name: LockWalletCheckout :one
SELECT * FROM checkouts WHERE id = sqlc.arg(id)::UUID FOR UPDATE;

-- name: CompleteWalletCheckout :one
UPDATE checkouts SET status = 'completed',
    credited_transaction_id = sqlc.arg(credited_transaction_id)::UUID,
    completed_at = now()
WHERE id = sqlc.arg(id)::UUID AND status = 'pending'
RETURNING *;
