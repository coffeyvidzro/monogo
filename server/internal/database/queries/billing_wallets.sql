-- name: CreateBillingWallet :one
INSERT INTO wallets (
    organization_id,
    currency
)
SELECT
    organization.id,
    sqlc.arg(currency)
FROM organizations AS organization
WHERE organization.id = sqlc.arg(organization_id)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
ON CONFLICT (organization_id, currency) DO NOTHING
RETURNING *;

-- name: GetBillingWallet :one
SELECT wallet.*
FROM wallets AS wallet
JOIN organizations AS organization
  ON organization.id = wallet.organization_id
WHERE wallet.organization_id = sqlc.arg(organization_id)
  AND wallet.currency = sqlc.arg(currency)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
LIMIT 1;

-- name: GetBillingWalletByID :one
SELECT wallet.*
FROM wallets AS wallet
JOIN organizations AS organization
  ON organization.id = wallet.organization_id
WHERE wallet.id = sqlc.arg(id)
  AND wallet.organization_id = sqlc.arg(organization_id)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
LIMIT 1;

-- name: ListBillingWallets :many
SELECT wallet.*
FROM wallets AS wallet
JOIN organizations AS organization
  ON organization.id = wallet.organization_id
WHERE wallet.organization_id = sqlc.arg(organization_id)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
ORDER BY wallet.created_at DESC;

-- name: SetBillingWalletStatus :one
UPDATE wallets AS wallet
SET
    status = sqlc.arg(status),
    updated_at = NOW()
FROM organizations AS organization
WHERE wallet.id = sqlc.arg(id)
  AND wallet.organization_id = sqlc.arg(organization_id)
  AND organization.id = wallet.organization_id
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
RETURNING wallet.*;

-- name: GetWalletEventByOperationID :one
SELECT *
FROM wallet_events
WHERE operation_id = sqlc.arg(operation_id)
LIMIT 1;

-- name: ListWalletEvents :many
SELECT *
FROM wallet_events
WHERE wallet_id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
ORDER BY wallet_version DESC
LIMIT sqlc.arg(page_limit);

-- name: ListChargeWalletEvents :many
SELECT event.*
FROM wallet_events AS event
JOIN charges AS charge
  ON charge.id = event.charge_id
 AND charge.organization_id = event.organization_id
WHERE event.organization_id = sqlc.arg(organization_id)
  AND event.charge_id = sqlc.arg(charge_id)
ORDER BY event.charge_sequence ASC;
