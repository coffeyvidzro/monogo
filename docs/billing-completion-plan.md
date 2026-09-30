# Billing and payment completion plan

## Executive assessment

The billing domain has a strong foundation: subscription entitlement and prepaid
wallet money are deliberately separate, wallet mutations are ledger-backed and
idempotent, provider webhook signatures are verified, and payment attempts and
provider events have durable identities. The implementation is not yet ready to
accept production money, however. The HTTP checkout flow currently creates a
local payment attempt but never starts a provider payment or attaches the
provider payment identifier. Consequently, no real provider webhook can be
correlated to that attempt.

The safest path is to complete one end-to-end wallet top-up flow first, while
preserving the provider-neutral payment and checkout records. Subscription
renewal semantics should be decided before extending the same orchestration to
subscriptions.

## What exists today

### Sound foundations

- A subscription is an entitlement; a prepaid wallet funds managed telecom
  operations. Neither balance substitutes for the other.
- Wallet credits, debits, holds, captures, and releases use database
  transactions and operation IDs to support safe retries.
- Checkouts distinguish `subscription` from `wallet_topup` and snapshot an
  amount and currency before payment begins.
- Payment attempts are separate from checkouts, support retries, and enforce a
  single active attempt per checkout.
- Stripe and Paystack clients can initiate payments, verify settlement, and
  authenticate webhooks.
- Provider events are stored with a payload digest and can only transition once
  to processed.
- Subscription middleware checks both status and the current entitlement
  period rather than trusting status alone.

### Production blockers

#### P0: provider initiation is disconnected from checkout confirmation

`checkout.Service.Confirm` moves the checkout to `processing` and creates a
payment attempt, but it does not call `stripe.Client.CreateCheckoutSession` or
`paystack.Client.ChargeMobileMoney`. It also does not call
`payments.Service.AttachProviderPaymentID`. The existing provider clients and
mappers therefore have no production call site.

The confirmation response also lacks provider-specific data such as a Stripe
client secret or a Paystack authorization message. A real client cannot finish
the payment.

**Required change:** introduce a concrete payment orchestrator used by checkout
confirmation. It should:

1. claim or reuse the active local payment attempt;
2. create the provider request with the checkout reference as the provider
   idempotency/reference key;
3. persist the provider payment ID before returning success;
4. map the provider response into the checkout's next action;
5. return only the public provider data needed by the client.

Do not hold a database transaction open during an external HTTP request. Use a
short database claim, make the provider request with a stable idempotency key,
then use a short compare-and-set transaction to attach the response.

#### P0: fulfillment and payment state are not committed atomically

Webhook processing currently performs these steps in separate transactions:

1. store the provider event;
2. mark the payment successful;
3. credit the wallet or activate the subscription;
4. mark the checkout successful;
5. mark the event processed.

The wallet operation ID prevents duplicate credits, but a crash or concurrent
delivery can still leave contradictory visible states and repeated conflicts.
Subscription activation and checkout completion have a similar race.

**Required change:** after remote verification, lock the provider event,
payment, and checkout and perform the local settlement in one database
transaction. Wallet credit or subscription activation, payment success,
checkout success, and event processing must commit together. Remote provider
verification must happen before, not during, that transaction.

#### P0: successful late payments can be stranded

The expiration job expires both pending and processing checkouts. A provider
may settle after the local 30-minute deadline. The webhook can then mark its
payment successful, while checkout completion rejects the expired checkout and
never credits the wallet or activates the subscription.

**Required change:** define an explicit late-settlement policy. The recommended
policy for top-ups is to honor a cryptographically verified successful payment
exactly once even if the local checkout expired. If a product cannot be
fulfilled, create a refund/review task rather than silently retaining funds.
Pending checkouts may expire normally; processing expiration must account for
the provider's terminal state.

#### P0: subscriptions are one-time activations, not recurring billing

A successful subscription checkout activates exactly one locally calculated
month. No renewal invoice/payment is created, no provider subscription identity
is stored, and no job moves an ended period to `past_due` or `cancelled`.

Before implementation, choose one model:

- **Provider-managed recurring billing (recommended for cards):** store provider
  customer, subscription, price, and invoice identities; advance entitlement
  only from verified invoice settlement events.
- **Monogo-managed monthly invoices:** create a local invoice and checkout each
  period, charge an explicitly authorized reusable payment method, and advance
  entitlement only after settlement.

Do not model a recurring subscription as repeated anonymous checkout sessions.

#### P1: failure, cancellation, and asynchronous events are incomplete

Only successful provider events affect state. Failed, expired, cancelled, and
refunded provider events are recorded and then treated as processed without
updating the payment or checkout. Stripe asynchronous failure and Paystack
failure events therefore cannot close attempts. Refund status constants exist
in Go but are not accepted by the payment table constraint and there is no
refund ledger workflow.

**Required change:** define an allowlisted provider event matrix. Each event
must map to an idempotent domain command: succeed, fail, cancel, refund, ignore,
or quarantine. Unknown event types should be observably ignored, not mistaken
for completed business processing.

#### P1: provider reconciliation data is incomplete

Settlement verification validates amount and currency, which is good, but uses
the application clock as `paid_at`. The models do not retain provider customer,
fee, settlement timestamp, payment instrument summary, or provider request ID.

**Required change:** use the provider's settlement timestamp and persist a
small normalized reconciliation snapshot while retaining the immutable raw
event. Never store PAN, CVV, OTP, or other sensitive authentication material.

#### P1: webhook operations need hardening

- The body reader truncates payloads at 1 MiB instead of rejecting an oversized
  request.
- Webhook processing is synchronous, so provider/API or database latency can
  cause delivery retries and request pileups.
- There is no visible retry/dead-letter worker for unprocessed provider events.
- Operational metrics and alerts for unprocessed events, settlement failures,
  mismatched amounts, and late payments are missing.

**Required change:** reject oversized payloads, durably ingest authenticated
events quickly, and process them with a bounded retry worker. Alert on event age
and repeated terminal errors.

## Recommended target model

Keep the existing aggregate boundaries and add the minimum missing records:

- `provider_customers`: organization-to-provider customer identity.
- `provider_payment_methods`: token/reference plus non-sensitive display data
  and reusable-authorization status.
- `invoices`: immutable commercial obligation, period, amount, currency, and
  status. A checkout pays an invoice or funds a wallet; it should not itself be
  the long-lived billing obligation.
- `payment_refunds`: provider refund identity, amount, status, reason, and the
  compensating wallet ledger operation where applicable.
- optional `payment_processing_failures` or retry columns on provider events for
  worker scheduling and operator visibility.

Continue to use concrete services in constructors. Add small consumer-side
interfaces only at the checkout orchestrator boundary if tests need provider
polymorphism; provider packages should continue returning concrete response
structs.

## Delivery sequence

### Milestone 1: production-safe wallet top-up

1. Add provider initiation orchestration and public checkout response fields.
2. Pass stable idempotency/reference keys to both providers.
3. Atomically settle payment, wallet ledger credit, checkout, and provider
   event.
4. Implement late-success behavior and oversized-body rejection.
5. Add integration tests covering duplicate confirm, duplicate webhook,
   concurrent webhook delivery, crash/retry boundaries, amount mismatch, and
   success after expiration.
6. Add event retry processing, metrics, structured logs, and operator queries.

This milestone should launch behind per-provider and per-organization feature
flags with conservative top-up limits.

### Milestone 2: subscription lifecycle

1. Decide provider-managed versus Monogo-managed renewal per payment method.
2. Add invoices and provider customer/subscription identities.
3. Implement renewal success, failure, grace-period, cancellation-at-period-end,
   and plan-change state machines.
4. Derive entitlement periods from paid invoice periods, not webhook receipt
   time.
5. Test out-of-order events, duplicate invoices, failed renewal recovery, and
   cancellation races.

### Milestone 3: refunds and reconciliation

1. Add full and partial refund state machines.
2. Define wallet clawback behavior when top-up funds were already consumed;
   never make an implicit destructive adjustment.
3. Reconcile provider settlements against local payments daily.
4. Add a review queue for unmatched, overpaid, underpaid, disputed, and late
   transactions.

### Milestone 4: broader payment-method support

Complete Paystack OTP/phone continuation only if required by the selected
charge APIs. Add new methods behind provider capability checks rather than
growing checkout conditionals indefinitely.

## Definition of done for accepting money

- Every provider creation call has a stable idempotency key.
- Every successful provider payment maps to exactly one local payment.
- A local success produces exactly one entitlement or wallet ledger movement.
- Duplicate, concurrent, delayed, and out-of-order webhooks are safe.
- No successful payment can remain indefinitely without fulfillment, refund,
  or an alerted review item.
- Amount and currency are verified against a server-side snapshot.
- Subscription access follows a paid billing period and has explicit renewal
  and grace-period behavior.
- Refunds and disputes have explicit accounting behavior.
- Operators can inspect and retry unprocessed events without editing financial
  rows manually.
- End-to-end sandbox tests pass for Stripe and Paystack, including failure and
  timeout paths.

## Decisions needed before implementation

1. Are subscriptions provider-managed recurring products, or will Monogo issue
   and collect monthly invoices?
2. Which legal entity, countries, currencies, and payment methods launch first?
3. Should a verified late wallet top-up always be credited, or can policy
   require automatic refund above a lateness threshold?
4. What grace period applies after subscription renewal failure?
5. How should refunds behave after prepaid credit has already funded telecom
   usage?
6. What top-up limits, velocity limits, and manual-review thresholds are
   required for launch?

The first engineering slice should be narrow: one currency, Stripe card wallet
top-up, and the complete retry/concurrency test matrix. Once that path is
observable and reconcilable, Paystack mobile money can reuse the same local
settlement state machine.
