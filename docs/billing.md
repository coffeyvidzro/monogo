# Billing model

Monogo has two independent customer billing obligations:

1. **Subscription** pays for access to the Monogo platform.
2. **Prepaid PAYG** pays for Monogo-managed telecom products.

They must remain separate in storage, authorization, payment intent, and
customer-facing balances.

## Core rule

An active subscription answers:

> Is this organization allowed to use Monogo?

The prepaid wallet answers:

> Does this organization have enough funded balance for this managed operation?

A wallet balance never activates a subscription. A subscription never provides
managed carrier credit.

## Access matrix

| Subscription | Wallet | BYOC | Managed |
| --- | --- | --- | --- |
| active | funded | allowed | allowed |
| active | empty | allowed | blocked |
| inactive | funded | blocked | blocked |
| inactive | empty | blocked | blocked |

Commercial recovery endpoints remain available while service access is blocked
so an organization can pay its subscription, inspect its wallet, or add funds.

## Subscription

`subscription_plans` defines the monthly platform plans Monogo sells.

`subscriptions` records the organization's current plan and entitlement
period.

The subscription is paid independently of the prepaid wallet. Subscription
payments must not debit `wallets`.

The initial subscription states are:

- `pending`
- `active`
- `past_due`
- `cancelled`

New platform operations require an active subscription.

## BYOC

BYOC requires an active Monogo subscription.

The customer pays its carrier directly, so Monogo does not debit the prepaid
wallet for carrier usage.

The initial BYOC rule is therefore:

```
active subscription
        |
        +-- no --> reject
        |
        +-- yes --> allow
```

Any future Monogo-specific BYOC usage fee must be introduced explicitly as a
separate commercial product. It must not be implied by carrier usage.

## Prepaid PAYG

Managed products consume the organization's prepaid wallet.

The initial managed products are:

- managed voice
- outbound SMS
- outbound WhatsApp
- number purchase
- number renewal

The prepaid money model consists of:

- `wallets`: current settled balance and reserved amount
- `wallet_holds`: temporary authorization before an operation settles
- `wallet_ledger_entries`: immutable credits and debits

Available balance is:

```
balance_micros - reserved_micros
```

An operation must never authorize against the raw balance while ignoring
reserved funds.

## Wallet top-up

A successful wallet top-up creates a wallet credit.

```
payment provider
      |
      v
wallet top-up confirmed
      |
      v
wallet ledger credit
      |
      v
wallet balance increases
```

Payment-provider integration is not the source of wallet truth. The immutable
wallet ledger is.

Top-up payment intent and subscription payment intent are separate operations,
even when they use the same payment provider.

## Managed fixed-price operation

For fixed-price products such as SMS:

```
active subscription?
      |
      v
resolve customer retail price
      |
      v
enough available wallet balance?
      |
      v
create hold
      |
      v
perform provider operation
   /        \
success    failure
   |          |
capture     release
   |
wallet debit
```

Capturing a hold creates exactly one immutable debit for its operation ID.
Releasing a hold does not create a debit.

Retries must reuse the same operation ID so they cannot charge the customer
twice.

## Managed voice

Voice follows the same commercial model but needs separate credit-control
behavior because the final duration is not known when the call starts.

The billing foundation only requires:

1. an active subscription;
2. a customer-facing voice rate;
3. sufficient prepaid authorization before Monogo creates carrier exposure;
4. one final customer debit for the billable usage.

Real-time call balance extension and forced hangup are voice credit-control
features. They are not part of the foundational subscription/wallet model and
must not introduce an arbitrary fixed call-duration rule into the core billing
model.

## Customer retail pricing

Customer-facing prices are separate from wallets.

`voice_rates` contains managed voice retail prices.

`product_rates` contains fixed-unit retail prices for managed non-voice
products.

Pricing answers:

> What should Monogo charge this customer?

The wallet answers:

> Does the customer have enough prepaid money, and what money moved?

These responsibilities must not be merged.

## Phase 1 source of truth

| Responsibility | Source of truth |
| --- | --- |
| Platform plan catalog | `subscription_plans` |
| Platform entitlement | `subscriptions` |
| Current prepaid balance | `wallets` |
| Temporary prepaid authorization | `wallet_holds` |
| Settled prepaid money movement | `wallet_ledger_entries` |
| Managed voice retail pricing | `voice_rates` |
| Managed non-voice retail pricing | `product_rates` |

## Out of scope for the billing foundation

Supplier accounting is intentionally separate from customer billing.

The following are not required to make Subscription + Prepaid PAYG work:

- provider CDR accounting
- provider charge reconciliation
- gross-margin reporting
- payment gateways
- real-time voice OCS
- provider invoice ingestion

Those capabilities can be layered on after customer entitlement, retail
pricing, wallet authorization, and settlement are stable.

## Implementation order

1. Subscription entitlement.
2. Wallet credit, hold, capture, and release.
3. Retail rate resolution.
4. Fixed-price managed charging.
5. Managed voice settlement.
6. Stripe/Paystack subscription payments and wallet top-ups.
7. Supplier accounting and reconciliation.
