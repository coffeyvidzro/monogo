# Multi-carrier routing and number lifecycle

This increment establishes the safety boundaries for the next two product
phases. It deliberately keeps provider credentials and resource identifiers
inside the Leamout control plane.

## Multi-carrier orchestration

The normalized routing snapshot contains an integer per-minute rate plus ASR,
ALOC, latency, packet loss, health, and observation time. `RankCarriers` first
rejects unhealthy, stale, malformed, or below-floor candidates. It then adds
bounded quality penalties to the wholesale rate and returns a deterministic
primary/secondary/tertiary order. The runtime can move to the next already
ranked route immediately when connection setup fails or the provider returns a
retryable `5xx`; authentication, authorization, and most `4xx` responses must
not be retried onto another carrier.

Rate and telemetry ingestion remain separate from the hot path. Collectors
write effective-dated USD rate rows and upsert the current aggregate telemetry
snapshot. Managed route resolution applies longest-prefix pricing, ranks the
result once, writes an immutable decision and candidate audit in one
transaction, and returns the ordered alternatives. Missing or stale data fails
closed rather than silently bypassing quality or price policy.

## Number provisioning and lifecycle

`number_lifecycle_operations` is the durable idempotency and reconciliation
boundary for assignment, port-in, port-out, and release. Provider submission
is never inferred from an HTTP timeout: ambiguous results stay submitted and
are reconciled by provider reference before another external mutation occurs.
The operation record is distinct from the resulting number so failed and
manual-review attempts remain auditable.

Emergency-service mappings are versioned in `emergency_registrations`. Only
one pending/validating/active mapping may exist for a phone number. Activation
requires a provider reference and timestamp; replacing an address therefore
means deactivating the old registration and creating a new one, never editing
regulatory history in place.

## Next integration slices

1. Add carrier-specific collectors that populate the normalized rate and
   telemetry tables.
2. Feed the returned ranked alternatives to the SIP transaction layer and restrict failover
   to transport errors, timeouts, and configured retryable `5xx` responses.
3. Add provider adapters and authenticated APIs for porting, release, and E911
   validation, using lifecycle operation idempotency keys for every mutation.
4. Add messaging provider capability/rate snapshots to the same policy model;
   delivery receipts should remain asynchronous and idempotent.
5. Add fraud limits, prepaid authorization, and complete decision/outcome audit
   events before enabling managed multi-carrier traffic in production.
