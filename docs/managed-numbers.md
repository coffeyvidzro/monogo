# Managed DID acquisition

DIDWW is the initial Leamout-managed DID inventory, acquisition and inbound
provider. CommPeak supplies Leamout-managed outbound termination and optional
origination; it is not the DID acquisition provider. BYOC remains independent.

## Implemented

`GET /v1/numbers/available?contains=5551234` searches DIDWW with Leamout-owned
credentials when `DIDWW_API_KEY` is configured. Results are **display-only**:
no upstream IDs, SKUs, prices, or reservations are exposed.

The purchase-intent foundation in `027_create_managed_number_orders.sql`
and `managed_number_orders.sql` persists the organization, selected inventory,
accepted internal quote, idempotency key and request hash, financial reservation
reference, provider order and DID identities, and lifecycle state. It does not
call DIDWW or charge a wallet.

## Required orchestration before enabling a purchase endpoint

1. Resolve a server-generated, expiring selection for the authenticated
   organization. Verify DIDWW inventory availability, SKU, country and E.164
   number, and a trusted price and currency. Reject customer-supplied amounts.
   Hash all immutable purchase inputs, including selection, accepted price,
   currency, billing period, and quote, to bind the idempotency key.
2. Create the intent with a fresh UUID. On a zero-row insert, fetch by the
   organization-scoped idempotency key: replay only when its request hash and
   immutable request values match. Otherwise return a conflict. If another
   unresolved intent owns the number, do not issue an upstream order.
3. Reserve funds through a real prepaid ledger using a unique reservation
   reference for this intent. Only then mark the intent ready. The reservation
   UUID alone does not establish that funds were secured.
4. Atomically claim the ready intent before requesting the DIDWW order. Use
   the durable intent UUID as the DIDWW external reference; it is a recovery
   identifier, **not** provider-enforced idempotency. Never hold a database
   transaction open across the provider request.
5. If a provider response is lost, retain an unresolved state and reconcile
   the original order via its ID, external reference and independently verified
   DID ownership. An empty lookup is not permission for an automatic retry.
   Verified callbacks may trigger reconciliation but cannot activate a number.
6. After confirming acquisition, configure the selected DID to an approved
   Leamout-controlled DIDWW inbound SIP trunk, then independently verify the
   provider's DID/trunk relationship.
7. Create or update the managed phone-number record in the correct tenant and
   platform carrier scope, reconcile the prepaid capture, and activate only
   when ownership and inbound routing are verified. A number can remain
   unbound to a customer voice application without routing to arbitrary SIP
   destinations. CommPeak termination is not an activation prerequisite.

## Purchase-intent state

`pending_funding -> ready -> submitting -> provider_pending -> configuring`
is the intended progression. `outcome_unknown` must be reconciled before any
re-submission. `manual_review` is reserved for unresolved provider outcomes;
only confirmed pre-submission failures may use the initial `failed` query.
`completed` is reserved for verified provider ownership, inbound routing,
a linked managed phone number and correct financial settlement.

The initial partial unique indexes intentionally keep completed acquisitions
claimed. Number release and later re-acquisition must update these historical
claims explicitly in a subsequent lifecycle migration; do not drop the
uniqueness safeguard to work around a conflict.

## Not enabled by this PR

The schema and internal sqlc queries are foundations only. No public purchase
endpoint, trusted quote issuer, actual wallet reservation/capture, DIDWW order
submission, reconciliation worker, provider routing operation, number activation
or managed release is introduced here. The existing BYOC and inventory APIs
remain unchanged.
