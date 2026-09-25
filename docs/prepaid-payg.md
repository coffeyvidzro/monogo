# Prepaid PAYG wallet foundation

This change is stacked on the managed-number purchase PR, but is a separate
financial primitive. It adds only `wallets` and `wallet_transactions` and
an internal service for posting successful credits/debits. It does **not**
create a `wallet_topups` table.

- A wallet is unique per organization and three-letter currency. All amounts
  are positive 64-bit integers in the currency's minor units; no floats.
- A balance change requires a single transaction that locks the wallet,
  resolves an existing business reference, checks balance and overflow,
  updates the balance, and inserts an immutable ledger row before commit.
- `(wallet_id, reference_type, reference_id)` is unique across credits and
  debits; a retry with the same reference and identical content returns the
  original entry without changing the balance.
- Wallet credits are **not** proof of payment. A payment integration must
  validate successful provider settlement, amount, currency and payee before
  calling the internal Post method using a stable internal payment UUID.
  Pending/failed top-ups are tracked in the payment domain and do not enter
  the successful wallet ledger.
- A refund must create a new, separately referenced credit; a worker must
  verify the original debit and refund eligibility. Ledger entries cannot be
  updated or deleted, including by accidental cascades.
- Uncertain database COMMIT responses must be reconciled with the original
  reference; never retry using a new business reference.

## Integration gates

The parent PR's DIDWW purchase handler **currently submits upstream orders
without checking a wallet**. The managed purchase route must be held back
until a trusted price and currency, prepaid balance check, a unique debit,
a provisioning number and a durable purchase intent are committed as one
transaction *before* calling DIDWW. A generic internal credit method must
never be exposed as a customer-controlled balance top-up endpoint.

No payment integration, public wallet API, purchase debit, number provisioning
change or recurring renewal is connected in this stacked PR.
