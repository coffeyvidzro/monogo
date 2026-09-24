# Managed DID acquisition

DIDWW is the initial Monogo-managed DID inventory, acquisition, and inbound
provider. CommPeak remains independent outbound termination infrastructure.
Managed-number purchasing intentionally has **no wallet, stored-value balance,
or prepaid PAYG ledger** in this phase.

## API workflow

1. `GET /v1/numbers/available?contains=5551234` returns display-only E.164
   inventory. Provider resource and SKU identifiers are never exposed.
2. `POST /v1/numbers/managed` accepts `number` and `country_code` and requires
   an `Idempotency-Key`. The server re-resolves exact DIDWW inventory and its
   SKU immediately before creating a durable order. A global partial unique
   index prevents a second live order for either the E.164 number or provider
   inventory item.
3. The API atomically claims the order before the provider request. Once a
   request may have reached DIDWW, any error becomes `outcome_unknown`; the
   request is never blindly repeated. The response is `202 Accepted` and its
   `Location` points at `GET /v1/numbers/orders/{order_id}`.
4. Reconciliation locates the existing order by its provider ID or the durable
   Monogo order UUID used as DIDWW's external reference. It verifies that
   reference, waits for completion, then independently resolves the exact
   owned DID.
5. The reconciler resolves the single active, inbound-enabled DIDWW platform
   connection and its database-backed `voice_in_trunk` provider resource. It
   assigns the DID to that trunk and reads the DID back. Only a matching number,
   DID identity, and `voice_in_trunk` relationship permit activation.
6. Activation creates the managed `phone_numbers` row and completes the order
   in one serializable transaction. Normal inbound routing then uses the
   existing active-number and voice-binding lookup; an unbound number cannot
   route to an arbitrary destination.

## Durable lifecycle

`ready -> submitting -> provider_pending -> configuring -> completed` is the
normal path. `outcome_unknown` is reconciliation-only. Identity contradictions
move the order to `manual_review`; only a confirmed failure before submission
may release a number claim with `failed`.

The order stores verification timestamps, reconciliation scheduling and attempt
counts, provider identities, inbound trunk identity, and the activated phone
number. Completed acquisitions continue to hold their uniqueness claims.

## Configuration

Set the platform-owned `DIDWW_API_KEY`. The future backoffice must persist one
active, inbound-enabled DIDWW platform carrier connection and associate its
approved DIDWW Voice In Trunk through `carrier_connection_provider_resources`.
Missing or ambiguous database routing configuration fails managed purchases
closed while BYOC remains available.
