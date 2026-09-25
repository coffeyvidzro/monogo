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

-- name: ListPrepaidWallets :many
SELECT *
FROM wallets
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY currency, id;

-- name: GetPrepaidWalletByID :one
SELECT *
FROM wallets
WHERE id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
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

-- name: LockPrepaidWalletByID :one
SELECT *
FROM wallets
WHERE id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
FOR UPDATE;

-- name: GetWalletTransactionByReference :one
SELECT *
FROM wallet_transactions
WHERE wallet_id = sqlc.arg(wallet_id)
  AND reference_type = sqlc.arg(reference_type)
  AND reference_id = sqlc.arg(reference_id)
LIMIT 1;

-- name: GetWalletTransaction :one
SELECT * FROM wallet_transactions WHERE id = sqlc.arg(id) LIMIT 1;

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

-- Reservation operations update posted and reserved amounts while holding the
-- wallet row lock. Both previous values protect against stale writes.
-- name: SetPrepaidWalletAmounts :one
UPDATE wallets
SET balance_minor = sqlc.arg(balance_minor),
    reserved_minor = sqlc.arg(reserved_minor)
WHERE id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
  AND balance_minor = sqlc.arg(previous_balance_minor)
  AND reserved_minor = sqlc.arg(previous_reserved_minor)
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

-- name: CreateWalletReservation :one
INSERT INTO wallet_reservations (
    wallet_id, organization_id, amount_minor, operation_type, operation_id, expires_at
) VALUES (
    sqlc.arg(wallet_id), sqlc.arg(organization_id), sqlc.arg(amount_minor),
    sqlc.arg(operation_type), sqlc.arg(operation_id), sqlc.arg(expires_at)
)
ON CONFLICT (organization_id, operation_type, operation_id) DO NOTHING
RETURNING *;

-- name: GetWalletReservationByOperation :one
SELECT * FROM wallet_reservations
WHERE organization_id = sqlc.arg(organization_id)
  AND operation_type = sqlc.arg(operation_type)
  AND operation_id = sqlc.arg(operation_id)
LIMIT 1;

-- name: GetWalletReservation :one
SELECT * FROM wallet_reservations
WHERE id = sqlc.arg(id) AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: LockWalletReservation :one
SELECT * FROM wallet_reservations
WHERE id = sqlc.arg(id) AND organization_id = sqlc.arg(organization_id)
FOR UPDATE;

-- name: ExtendWalletReservation :one
UPDATE wallet_reservations
SET amount_minor = sqlc.arg(amount_minor), expires_at = sqlc.arg(expires_at)
WHERE id = sqlc.arg(id) AND status = 'active' AND expires_at > now()
RETURNING *;

-- name: ReleaseWalletReservation :one
UPDATE wallet_reservations
SET status = sqlc.arg(status),
    released_at = CASE WHEN sqlc.arg(status)::TEXT = 'released' THEN now() ELSE NULL END,
    expired_at = CASE WHEN sqlc.arg(status)::TEXT = 'expired' THEN now() ELSE NULL END
WHERE id = sqlc.arg(id) AND status = 'active'
  AND sqlc.arg(status)::TEXT IN ('released', 'expired')
RETURNING *;

-- name: CaptureWalletReservation :one
UPDATE wallet_reservations
SET status = 'captured', captured_amount_minor = sqlc.arg(captured_amount_minor),
    captured_transaction_id = sqlc.arg(captured_transaction_id), captured_at = now()
WHERE id = sqlc.arg(id) AND status = 'active'
RETURNING *;

-- name: ListExpiredWalletReservations :many
SELECT * FROM wallet_reservations
WHERE status = 'active' AND expires_at <= now()
ORDER BY expires_at, id
LIMIT sqlc.arg(row_limit);

-- name: ListWalletTransactionsByWalletID :many
SELECT wt.*
FROM wallet_transactions AS wt
JOIN wallets AS w ON w.id = wt.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.id = sqlc.arg(wallet_id)
ORDER BY wt.created_at DESC, wt.id DESC
LIMIT sqlc.arg(row_limit);
