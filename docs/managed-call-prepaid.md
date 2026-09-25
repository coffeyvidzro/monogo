# Managed telecom pay-as-you-go pricing and prepaid collection

Leamout sells raw managed telecom connectivity at customer-facing pay-as-you-go
tariffs derived from verified wholesale carrier costs plus an approved markup.
Prepaid is the funding method, **not** a fixed-duration subscription or package.

## Commercial pricing and usage

The `commercial/pricing` domain creates a customer tariff from a wholesale
USD-per-minute rate and a policy markup in basis points. Markup is calculated
**over wholesale cost**, not as a percentage of the final selling price.
Customer billing increments and minimum billable duration are explicit.
The pricing calculation uses integer micros and rounds charges to whole USD
cents; actual seconds, not the reservation's maximum duration, determine the
final customer charge.

For example, a $0.10/min wholesale rate plus 20% markup yields a $0.12/min
customer tariff. With 60-second increments, a confirmed 61-second call bills
two minutes ($0.24), while a confirmed 10-second call bills one minute ($0.12).
This illustrative tariff is not a production price.

Wholesale cost is estimated separately. Realized gross margin cannot be
determined from a retail tariff alone: provider-specific increments, connection
fees, currency, taxes, and authoritative carrier CDR costs must be reconciled.

## Prepaid authorization

`wallets.Service.AuthorizeManagedCall` accepts a customer-facing pricing
quote and an independently enforced maximum call duration. It reserves the
highest customer-rated charge for that duration, using the durable
`organization_id + managed_call + call_id` idempotency key. A higher-cost
failover carrier requires authorization against the approved policy and
remaining reserved amount before origination. This is a spending cap, **not**
a sale of the maximum duration.

The wallet is not the source of tariffs. A trusted commercial service must
verify wholesale sources and approved markup policy and persist an immutable
call-level quote before permitting managed origination. A caller-provided
quote, even if its markup arithmetic is valid, does not prove policy approval.

## Actual usage and settlement

`wallets.Service.SettleManagedCall` now derives the amount to capture by
rating confirmed actual seconds against the quoted customer tariff, rather
than accepting a caller-supplied final charge. It requires a final billing
evidence reference and settles through the idempotent wallet transaction
keyed to the logical call ID. Zero-usage release requires verified evidence
of no billable carrier usage. Expired or ambiguous managed-call holds cannot
be released by the generic wallet expiry worker or generic release API.

The current settlement method does not yet authenticate the CDR, enforce
that its quote is the same immutable quote used for authorization, or store
the carrier's actual settled cost. All three checks are required at a
trusted call-rating and reconciliation boundary before enabling billing.

## Remaining production controls

- Persist the wholesale source, currency, effective date, approved markup,
  original customer quote, selected carrier, and final billing evidence.
- Enforce a bounded call duration or renewable spending window at the SIP
  and media runtime, including process crashes and failover.
- Connect verified provider CDRs, destination-specific billing rules, and
  final wholesale cost to customer usage rating and idempotent settlement.
- Handle failed, ambiguous, zero-charge, exhausted-wallet, and replayed
  calls with production database and SIP integration tests.

Until these controls are enforced, managed outbound origination remains
blocked. BYOC calls are unaffected.
