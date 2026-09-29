# Billing model

Monogo separates platform entitlement from carrier consumption.

## Subscription

An active subscription grants an organization access to the platform. Both
BYOC and managed calling require the subscription to be active for the current
billing period. A prepaid wallet balance does not replace this requirement.

Calls that are already in progress are not terminated merely because their
organization's subscription period ends. The subscription check applies when
a new inbound or outbound call is admitted.

## BYOC

BYOC uses a carrier account supplied and paid for by the customer. It requires
an active Monogo subscription, but Monogo does not debit the prepaid wallet for
BYOC carrier usage.

The customer's carrier may bill the customer independently. “No Monogo PAYG
carrier charge” does not mean that the upstream carrier service is free.

## Managed carrier service

Managed carrier service requires both:

1. an active Monogo subscription; and
2. enough available prepaid wallet balance for the operation.

For managed outbound voice, Monogo resolves the applicable customer-facing
retail rate and reserves prepaid funds before originating the call. If
origination fails before the call is answered, the reservation is released. If
the call is answered, the reservation is captured as an immutable wallet
ledger debit.

Customer-facing voice rates are distinct from upstream wholesale costs. The
difference must remain visible for reconciliation and margin reporting.

## Access matrix

| Mode | Active subscription | Prepaid carrier balance | Monogo carrier PAYG |
| --- | --- | --- | --- |
| BYOC | Required | Not required | No |
| Managed | Required | Required | Yes |

Billing, top-up, renewal, and account-recovery operations must remain available
when commercial service access is restricted. A customer must not be prevented
from resolving the condition that caused the restriction.

## Implementation status

Organization-scoped platform, AI, and telecom HTTP APIs enforce the active
subscription requirement. Commercial APIs for plans, subscriptions, wallet
visibility, and future top-up workflows are deliberately exempt so an
organization can restore access.

Managed inbound and outbound voice reserve ten started minutes at admission,
enforce that authorization as the maximum call duration, and settle only the
started minutes actually consumed. Unused authorization is released.

Managed SMS and WhatsApp submissions reserve their configured fixed retail
price before contacting the platform carrier. A provider rejection releases
the hold; provider acceptance or an unknown provider outcome captures it.

Managed number purchases reserve both the configured acquisition price and the
first renewal period before an upstream order is submitted. The charge is
captured only after ownership and routing are verified and the number is
activated. Subsequent renewal periods are authorized and captured automatically by the
managed-number renewal worker.

Fixed-unit managed-product prices are stored in `product_rates`. Rates may be
global or organization-specific, are effective-dated, and support exact
two-letter country selectors with a `*` global fallback. Per-started-minute
voice prices are stored separately in `voice_rates` because voice resolution
uses the longest matching telephone prefix and inbound/outbound direction.

## Managed-number renewals

A managed number receives a monthly renewal anchor when activation succeeds.
The worker creates one durable renewal record per number and billing period,
resolves the effective `number_renewal` retail price, and captures that amount
from the prepaid wallet. The billing operation identifier is stable for the
period, so worker retries cannot produce duplicate customer debits. Insufficient
funds and transient pricing failures remain visible as `payment_failed` renewal
records and are retried with capped exponential backoff.

## Provider costs and margin reporting

Actual supplier costs are immutable evidence, not inferred from Monogo retail
rates. `provider_costs` records the provider, its unique invoice or usage record,
the managed product, currency, amount, incurred time, and raw source payload.
Each cost must reference the exact captured wallet operation that generated the
customer revenue. Replaying identical provider evidence is idempotent; changing
an existing provider record is rejected.

The `managed_margin_entries` view reports captured managed revenue, recorded
provider cost, and gross profit per operation. Provider adapters and invoice
importers must record costs from provider-issued evidence; missing costs remain
zero in the view so reconciliation can identify incomplete ingestion rather
than fabricating supplier expense from retail prices. Existing voice CDR and
wholesale-charge evidence remains the detailed source for voice cost accounting.

DIDWW acquisition reconciliation records the completed order amount and raw
order response automatically after the corresponding customer hold is captured.
Recurring supplier invoices and messaging usage reports use the same immutable
provider-cost recorder when their provider evidence is imported.
