# Managed call prepaid authorization and settlement

This implementation is a wallet-domain building block. Managed carrier
origination is still blocked in `telecom/calls/service.go` until the
remaining runtime and billing contracts below are enforced.

## Durable authorization

`wallets.Service.AuthorizeManagedCall` reserves an upper-bound amount in a
USD prepaid wallet, rounding micros to whole minor units conservatively.
`organization_id + managed_call + call_id` is the unique reservation
operation. Replays do not authorize a previously expired or settled hold.

Rates passed to this method **must** be the approved customer tariff,
including required margin, setup fees, carrier-specific billing increments,
and taxes where applicable; a raw wholesale route rate is not sufficient.
The maximum duration must be enforced by the SIP/media runtime, including
outbound failover and abrupt worker loss, before enabling managed origination.

## Retention during uncertain outcomes

An expired `managed_call` reservation remains held instead of being released
by the general expiration job. `wallets.Service.Expire` also refuses to
expire one directly. An unresolved call cannot be treated as free just
because its local timeout elapsed.

The reservation can be recovered by the organization and logical call ID.
An operator or recovery job must independently verify carrier termination
and the authoritative final billing result before settlement. No current
implementation automatically obtains such a result from DIDWW or CommPeak.

## Idempotent finalization

`SettleManagedCall` requires a nonempty billing evidence reference and an
explicit final amount in USD minor units. The caller must validate this
evidence against an authenticated provider CDR or other authoritative billing
record and confirm the associated call and organization. The method does
**not** independently authenticate, store, or deduplicate the evidence.

A positive amount is captured through the existing transactional wallet
service with `managed_call + call_id` as the immutable ledger reference.
Zero can be released only after explicit verification of no billable charge.
A final amount above the original hold is rejected for manual review; no
unreserved debit is attempted. Replayed captures with different amounts
conflict. The existing wallet transaction and reservation status are
committed together.

## Required before turning on managed origination

1. Persist an authoritative per-call pricing snapshot and billing evidence
   reference; enforce a provider-independent maximum billable exposure.
2. Enforce maximum call duration and reliable disconnect at the SIP/media
   boundary; handle failover and orphaned channels.
3. Configure continuous admission controls that recover after process loss,
   and ensure reservation duration cannot underfund active calls.
4. Connect verified, authenticated provider CDR reconciliation to
   `SettleManagedCall`, with an explicit terminal lifecycle and manual-review
   path for missing or contradictory outcomes.
5. Prove wallet exhaustion, double-submit, replay, unknown-outcome, expiry,
   and carrier failover behavior with database and SIP integration tests.

Until then, managed calls fail closed; customer-owned BYOC calling is
unchanged.
