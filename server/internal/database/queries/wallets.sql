-- name: CreatePrepaidWallet :one
INSERT INTO wallets (organization_id, currency)
SELECT o.id, sqlc.arg(currency)::TEXT
FROM organizations AS o
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (organization_id, currency) DO NOTHING
RETURNING *;

-- name: GetPrepaidWallet :one
SELECT w.*
FROM wallets AS w
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.currency = sqlc.arg(currency)
LIMIT 1;

-- Lock the wallet before resolving a repeated financial reference, computing
-- the next balance, and inserting the entry in the SAME database transaction.
--
-- name: LockPrepaidWallet :one
SELECT *
FROM wallets
WHERE organization_id = sqlc.arg(organization_id)
  AND currency = sqlc.arg(currency)
FOR UPDATE;

-- name: GetWalletTransactionByReference :one
SELECT *
FROM wallet_transactions
WHERE wallet_id = sqlc.arg(wallet_id)
  AND reference_type = sqlc.arg(reference_type)
  AND reference_id = sqlc.arg(reference_id)
LIMIT 1;

-- Only the internal wallet service should call this, after row locking and
-- only in the transaction that also inserts the corresponding ledger entry.
--
-- name: SetPrepaidWalletBalance :one
UPDATE wallets
SET balance_minor = sqlc.arg(balance_minor)
WHERE id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
  AND balance_minor = sqlc.arg(previous_balance_minor)
RETURNING *;

-- name: CreateWalletTransaction :one
INSERT INTO wallet_transactions (
    wallet_id,
    direction,
    reason,
    amount_minor,
    balance_after_minor,
    reference_type,
    reference_id
)
VALUES (
    sqlc.arg(wallet_id),
    sqlc.arg(direction),
    sqlc.arg(reason),
    sqlc.arg(amount_minor),
    sqlc.arg(balance_after_minor),
    sqlc.arg(reference_type),
    sqlc.arg(reference_id)
)
RETURNING *;

-- name: ListWalletTransactions :many
SELECT wt.*
FROM wallet_transactions AS wt
JOIN wallets AS w ON w.id = wt.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.currency = sqlc.arg(currency)
ORDER BY wt.created_at DESC, wt.id DESC
LIMIT sqlc.arg(row_limit);
