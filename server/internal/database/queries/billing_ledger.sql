-- name: GetWalletLedgerEntry :one
SELECT entry.*
FROM wallet_ledger_entries AS entry
JOIN wallets AS wallet
  ON wallet.id = entry.wallet_id
 AND wallet.organization_id = entry.organization_id
WHERE entry.id = sqlc.arg(id)
  AND entry.organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: ListWalletLedgerEntries :many
SELECT entry.*
FROM wallet_ledger_entries AS entry
JOIN wallets AS wallet
  ON wallet.id = entry.wallet_id
 AND wallet.organization_id = entry.organization_id
WHERE entry.wallet_id = sqlc.arg(wallet_id)
  AND entry.organization_id = sqlc.arg(organization_id)
ORDER BY entry.occurred_at DESC, entry.id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListChargeLedgerEntries :many
SELECT entry.*
FROM wallet_ledger_entries AS entry
JOIN charges AS charge
  ON charge.id = entry.charge_id
 AND charge.organization_id = entry.organization_id
WHERE entry.organization_id = sqlc.arg(organization_id)
  AND entry.charge_id = sqlc.arg(charge_id)
ORDER BY entry.occurred_at ASC, entry.id ASC;
