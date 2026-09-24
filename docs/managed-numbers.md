# Managed DID acquisition: initial scope

DIDWW is the initial Leamout-managed DID inventory, acquisition, and inbound
provider. CommPeak supplies Leamout-managed outbound termination and optional
origination; it is not the DID acquisition provider. BYOC remains independent
of both Leamout-managed providers.

## Implemented in this PR

`GET /v1/numbers/available?contains=5551234` searches DIDWW using
Leamout-owned credentials when `DIDWW_API_KEY` is configured. `contains` is
required and must contain 3–15 ASCII digits. Responses include only displayable
E.164 numbers, not DIDWW IDs, SKUs, carrier connections, prices, or reservations.
An unconfigured or unavailable provider returns an error rather than fake
inventory. This is a read-only inventory endpoint, **not a purchasable offer**.
Country filtering and pricing are not implemented in this slice.

## Not yet implemented

No managed-number purchase, prepaid reservation/capture, selection token,
provider order, inbound trunk assignment, activation, renewal, or provider release
is enabled by this PR. The existing BYOC API and managed-number release guard
are unchanged.

Before enabling purchases, implement customer pricing and prepaid funding,
a tenant-bound expiring selection, a durable unique purchase intent, reconciliation
of uncertain DIDWW order outcomes without blind retries, provider ownership and
inbound routing verification, and activation only after confirmation. CommPeak
termination must not be required for DIDWW number activation.
