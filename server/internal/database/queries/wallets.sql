-- name: CreateWallet :one
INSERT INTO wallets (
    organization_id,
    currency
)
SELECT
    o.id,
    sqlc.arg(currency)
FROM organizations AS o
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (organization_id, currency) DO NOTHING
RETURNING *;

-- name: GetActiveWalletByID :one
SELECT w.*
FROM wallets AS w
JOIN organizations AS o
  ON o.id = w.organization_id
WHERE w.id = sqlc.arg(id)
  AND w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: GetActiveWalletByOrganizationCurrency :one
SELECT w.*
FROM wallets AS w
JOIN organizations AS o
  ON o.id = w.organization_id
WHERE w.organization_id = sqlc.arg(organization_id)
  AND w.currency = sqlc.arg(currency)
  AND w.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: LockActiveWallet :one
SELECT w.*
FROM wallets AS w
JOIN organizations AS o
  ON o.id = w.organization_id
WHERE w.id = sqlc.arg(id)
  AND w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
FOR UPDATE OF w;

-- name: ApplyWalletBalance :one
UPDATE wallets AS w
SET
    balance_micros = w.balance_micros + sqlc.arg(delta_micros),
    updated_at = NOW()
WHERE w.id = sqlc.arg(id)
  AND w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND w.balance_micros + sqlc.arg(delta_micros) >= 0
  AND EXISTS (
      SELECT 1
      FROM organizations AS o
      WHERE o.id = w.organization_id
        AND o.status = 'active'
        AND o.deleted_at IS NULL
  )
RETURNING w.*;

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
JOIN organizations AS o
  ON o.id = w.organization_id
WHERE w.id = sqlc.arg(wallet_id)
  AND w.organization_id = sqlc.arg(organization_id)
  AND w.status = 'active'
  AND w.balance_micros = sqlc.arg(balance_after_micros)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: GetWalletLedgerEntryByOperationID :one
SELECT *
FROM wallet_ledger_entries
WHERE organization_id = sqlc.arg(organization_id)
  AND wallet_id = sqlc.arg(wallet_id)
  AND operation_id = sqlc.arg(operation_id)
LIMIT 1;

-- name: ListWalletLedgerEntries :many
SELECT *
FROM wallet_ledger_entries
WHERE organization_id = sqlc.arg(organization_id)
  AND wallet_id = sqlc.arg(wallet_id)
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: FreezeWallet :one
UPDATE wallets
SET
    status = 'frozen',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: ActivateWallet :one
UPDATE wallets AS w
SET
    status = 'active',
    updated_at = NOW()
WHERE w.id = sqlc.arg(id)
  AND w.organization_id = sqlc.arg(organization_id)
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
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('active', 'frozen')
  AND balance_micros = 0
RETURNING *;
