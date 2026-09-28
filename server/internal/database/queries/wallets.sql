-- name: CreateWallet :one
INSERT INTO wallets (
    organization_id
)
SELECT
    o.id
FROM organizations AS o
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (organization_id) DO NOTHING
RETURNING *;

-- name: GetActiveWalletByOrganization :one
SELECT w.*
FROM wallets AS w
JOIN organizations AS o
  ON o.id = w.organization_id
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: LockActiveWallet :one
SELECT w.*
FROM wallets AS w
JOIN organizations AS o
  ON o.id = w.organization_id
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
FOR UPDATE OF w;

-- Existing holds must still be captured or released if an organization or
-- wallet is later frozen. New spending continues to require LockActiveWallet.
-- name: LockWalletForSettlement :one
SELECT w.*
FROM wallets AS w
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status IN ('active', 'frozen')
FOR UPDATE;

-- name: ApplyWalletBalance :one
UPDATE wallets AS w
SET
    balance_micros = w.balance_micros + sqlc.arg(delta_micros),
    updated_at = NOW()
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND w.balance_micros + sqlc.arg(delta_micros) >= w.reserved_micros
  AND EXISTS (
      SELECT 1
      FROM organizations AS o
      WHERE o.id = w.organization_id
        AND o.status = 'active'
        AND o.deleted_at IS NULL
  )
RETURNING w.*;

-- name: ReserveWalletBalance :one
UPDATE wallets AS w
SET
    reserved_micros = w.reserved_micros + sqlc.arg(amount_micros),
    updated_at = NOW()
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND w.balance_micros - w.reserved_micros >= sqlc.arg(amount_micros)
  AND EXISTS (
      SELECT 1
      FROM organizations AS o
      WHERE o.id = w.organization_id
        AND o.status = 'active'
        AND o.deleted_at IS NULL
  )
RETURNING w.*;

-- name: CaptureWalletReservedBalance :one
UPDATE wallets AS w
SET
    balance_micros = w.balance_micros - sqlc.arg(amount_micros),
    reserved_micros = w.reserved_micros - sqlc.arg(amount_micros),
    updated_at = NOW()
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status IN ('active', 'frozen')
  AND w.balance_micros >= sqlc.arg(amount_micros)
  AND w.reserved_micros >= sqlc.arg(amount_micros)
RETURNING w.*;

-- name: ReleaseWalletReservedBalance :one
UPDATE wallets AS w
SET
    reserved_micros = w.reserved_micros - sqlc.arg(amount_micros),
    updated_at = NOW()
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status IN ('active', 'frozen')
  AND w.reserved_micros >= sqlc.arg(amount_micros)
RETURNING w.*;

-- name: CreateWalletHold :one
INSERT INTO wallet_holds (
    wallet_id,
    organization_id,
    operation_id,
    amount_micros,
    reason,
    reference_type,
    reference_id
) VALUES (
    sqlc.arg(wallet_id),
    sqlc.arg(organization_id),
    sqlc.arg(operation_id),
    sqlc.arg(amount_micros),
    sqlc.arg(reason),
    sqlc.narg(reference_type),
    sqlc.narg(reference_id)
)
RETURNING *;

-- name: GetWalletHoldByOperationID :one
SELECT *
FROM wallet_holds
WHERE organization_id = sqlc.arg(organization_id)
  AND operation_id = sqlc.arg(operation_id)
LIMIT 1;

-- name: MarkWalletHoldCaptured :one
UPDATE wallet_holds
SET
    status = 'captured',
    captured_at = NOW(),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND operation_id = sqlc.arg(operation_id)
  AND status = 'active'
RETURNING *;

-- name: MarkWalletHoldReleased :one
UPDATE wallet_holds
SET
    status = 'released',
    released_at = NOW(),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND operation_id = sqlc.arg(operation_id)
  AND status = 'active'
RETURNING *;

-- name: CreateWalletLedgerEntry :one
INSERT INTO wallet_ledger_entries (
    wallet_id,
    organization_id,
    operation_id,
    direction,
    reason,
    amount_micros,
    balance_after_micros,
    reference_type,
    reference_id,
    occurred_at
)
SELECT
    w.id,
    w.organization_id,
    sqlc.arg(operation_id),
    sqlc.arg(direction),
    sqlc.arg(reason),
    sqlc.arg(amount_micros),
    sqlc.arg(balance_after_micros),
    sqlc.narg(reference_type),
    sqlc.narg(reference_id),
    sqlc.arg(occurred_at)
FROM wallets AS w
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.balance_micros = sqlc.arg(balance_after_micros)
RETURNING *;

-- name: GetWalletLedgerEntryByOperationID :one
SELECT *
FROM wallet_ledger_entries
WHERE organization_id = sqlc.arg(organization_id)
  AND operation_id = sqlc.arg(operation_id)
LIMIT 1;

-- name: ListWalletLedgerEntries :many
SELECT *
FROM wallet_ledger_entries
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: FreezeWallet :one
UPDATE wallets
SET
    status = 'frozen',
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: ActivateWallet :one
UPDATE wallets AS w
SET
    status = 'active',
    updated_at = NOW()
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'frozen'
  AND EXISTS (
      SELECT 1
      FROM organizations AS o
      WHERE o.id = w.organization_id
        AND o.status = 'active'
        AND o.deleted_at IS NULL
  )
RETURNING w.*;

-- name: CloseWallet :one
UPDATE wallets
SET
    status = 'closed',
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND status IN ('active', 'frozen')
  AND balance_micros = 0
  AND reserved_micros = 0
RETURNING *;
