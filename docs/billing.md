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
activated. Automated authorization for subsequent renewal periods still needs
a renewal scheduler before indefinite managed-number service is production
ready.

Fixed-unit managed-product prices are stored in `product_rates`. Rates may be
global or organization-specific, are effective-dated, and support exact
two-letter country selectors with a `*` global fallback. Per-started-minute
voice prices are stored separately in `voice_rates` because voice resolution
uses the longest matching telephone prefix and inbound/outbound direction.
