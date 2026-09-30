# Billing model

Leamout is a subscription software platform. Customers bring and pay their own
carriers directly; Leamout does not resell telecom usage, phone numbers, or
messaging capacity.

## Commercial architecture

```text
Customer
   |
   +-- carrier / SIP trunk / numbers ------> carrier bills customer
   |
   +-- Leamout subscription --------------> Stripe
                                               |
                                               v
                                      platform entitlement
```

## Source of truth

- `subscription_plans` defines the USD monthly platform plans.
- `subscriptions` defines whether an organization is entitled to use Leamout.
- `checkouts` represents subscription purchase sessions.
- `payments` and `payment_provider_events` record Stripe attempts and
  authenticated provider events.

Stripe moves money. It is not the entitlement source of truth.

## BYOC

Carrier connectivity is customer-owned.

Customers configure their own carrier connections, SIP trunks, phone numbers,
and messaging connections. Their carrier bills them directly. Leamout does not
maintain prepaid telecom wallets, retail voice rates, product rates, managed
number purchases, or managed number renewals.

An active Leamout subscription answers one commercial question:

> Is this organization allowed to use the Leamout platform?

Carrier spend is outside Leamout's customer billing ledger.

## Currency

Leamout subscription billing is USD-only in v1 and is collected through Stripe
card payments.

## Runtime boundary

Telephony remains a core product capability. BYOC does not mean removing
carrier integrations, trunks, routing, SIP, phone-number bindings, WebRTC, or
media control. It means Leamout orchestrates customer-owned connectivity rather
than becoming the carrier of record.

This keeps the commercial model focused while engineering moves toward the
Agent Runtime: realtime sessions, barge-in, model adapters, tools, handoffs, and
observability.
