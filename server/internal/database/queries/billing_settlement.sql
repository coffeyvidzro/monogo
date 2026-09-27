-- name: LockOCSWallet :one
SELECT *
FROM wallets
WHERE id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
FOR UPDATE;

-- name: LockOCSOperation :exec
SELECT pg_advisory_xact_lock(
    hashtextextended(sqlc.arg(operation_id)::text, 0)
);

-- name: LockOCSCharge :one
SELECT *
FROM charges
WHERE id = sqlc.arg(charge_id)
  AND organization_id = sqlc.arg(organization_id)
  AND wallet_id = sqlc.arg(wallet_id)
FOR UPDATE;

-- name: CreateOCSWalletEvent :one
INSERT INTO wallet_events (
    wallet_id,
    organization_id,
    charge_id,
    operation_id,
    wallet_version,
    charge_sequence,
    event_type,
    balance_delta_micros,
    reserved_delta_micros,
    balance_after_micros,
    reserved_after_micros,
    charge_authorized_after_micros,
    charge_consumed_after_micros,
    charge_reserved_after_micros,
    charge_status,
    occurred_at
) VALUES (
    sqlc.arg(wallet_id),
    sqlc.arg(organization_id),
    sqlc.narg(charge_id),
    sqlc.arg(operation_id),
    sqlc.arg(wallet_version),
    sqlc.narg(charge_sequence),
    sqlc.arg(event_type),
    sqlc.arg(balance_delta_micros),
    sqlc.arg(reserved_delta_micros),
    sqlc.arg(balance_after_micros),
    sqlc.arg(reserved_after_micros),
    sqlc.narg(charge_authorized_after_micros),
    sqlc.narg(charge_consumed_after_micros),
    sqlc.narg(charge_reserved_after_micros),
    sqlc.narg(charge_status),
    sqlc.arg(occurred_at)
)
RETURNING *;

-- name: ApplyOCSWalletProjection :one
UPDATE wallets
SET
    balance_micros = sqlc.arg(balance_after_micros),
    reserved_micros = sqlc.arg(reserved_after_micros),
    ocs_version = sqlc.arg(wallet_version),
    updated_at = NOW()
WHERE id = sqlc.arg(wallet_id)
  AND organization_id = sqlc.arg(organization_id)
  AND ocs_version = sqlc.arg(previous_wallet_version)
RETURNING *;

-- name: ApplyOCSChargeProjection :one
UPDATE charges
SET
    status = sqlc.arg(charge_status),
    authorized_micros = sqlc.arg(authorized_micros),
    consumed_micros = sqlc.arg(consumed_micros),
    reserved_micros = sqlc.arg(reserved_micros),
    ocs_sequence = sqlc.arg(charge_sequence),
    closed_at = sqlc.narg(closed_at),
    updated_at = NOW()
WHERE id = sqlc.arg(charge_id)
  AND organization_id = sqlc.arg(organization_id)
  AND wallet_id = sqlc.arg(wallet_id)
  AND ocs_sequence = sqlc.arg(previous_charge_sequence)
RETURNING *;

-- name: CreateOCSLedgerEntry :one
INSERT INTO wallet_ledger_entries (
    wallet_event_id,
    wallet_id,
    organization_id,
    charge_id,
    direction,
    reason,
    amount_micros,
    balance_after_micros,
    occurred_at
) VALUES (
    sqlc.arg(wallet_event_id),
    sqlc.arg(wallet_id),
    sqlc.arg(organization_id),
    sqlc.narg(charge_id),
    sqlc.arg(direction),
    sqlc.arg(reason),
    sqlc.arg(amount_micros),
    sqlc.arg(balance_after_micros),
    sqlc.arg(occurred_at)
)
RETURNING *;
