# Multi-carrier orchestration and number provisioning lifecycle

This document describes the implementation on the current telecom stack.
It distinguishes enforced behavior from planned safety requirements.
Provider credentials and external resource identifiers stay inside Leamout's
control plane.

## Module boundaries

- `telecom/routing` selects authorized outbound carrier routes and validates
  inbound number bindings. It does not buy, port, or release telephone numbers.
- `telecom/calls` records call attempts and executes ordered outbound routes
  through `runtime/calling`.
- `telecom/numbers` owns BYOC number inventory, managed purchase orders,
  customer assignments, and verified inbound delivery configuration.
- `telecom/lifecycle` owns durable release, emergency-registration, and
  port-in operations, separate from the resulting number resource.
- `integrations/carriers/didww` handles DIDWW-specific provider requests.

An outbound carrier failover does not move a DID to another inbound hosting
provider. Inbound redundancy requires supported delivery destinations or a
separate provider-number migration or porting operation.

## Multi-carrier orchestration: implemented

The managed routing snapshot contains an integer USD-per-minute rate and
ASR, average length of call, latency, packet loss, health, and observation time.
`RankCarriers` rejects malformed, stale, unhealthy, and below-floor
candidates, then sorts all eligible routes deterministically using the
wholesale rate plus bounded quality penalties. The outbound executor considers
only the first three routes; ranking itself is not limited to three entries.
These routes may belong to the same carrier, so three attempts do not
necessarily establish provider-level diversity.

Routing decisions and candidate snapshots are written transactionally.
The decision's selected route can subsequently change after successful
failover; the candidate snapshots retain decision-time measurements.

The outbound executor advances on classified transport errors, carrier
capacity rejections, and SIP `5xx` responses. It does not retry validation
failures, other SIP responses, or originate timeouts whose remote call
establishment is uncertain. An error accompanied by an established channel ID
also stops failover. Outbound origination errors cannot, by themselves, prove
that no remote call exists; transport classification needs further validation
against the underlying SIP transaction layer.

Attempt persistence is currently best effort: the call service does not fail
the call if recording an individual attempt fails. Durable audit enforcement
is a remaining production requirement.

### Remaining orchestration work

1. Implement and validate carrier-specific rate and health collectors. The
   normalized SQL tables and routing queries do not collect live data by
   themselves.
2. Connect asynchronous SIP transaction outcomes to typed originate errors,
   distinguishing confirmed setup failure from ambiguous remote state.
3. Enforce prepaid authorization, fraud limits, spending limits, and allowed
   carrier policies before enabling managed multi-carrier traffic.
4. Verify provider-diverse alternatives under an explicit price and quality
   policy; endpoint redundancy is not automatically carrier redundancy.
5. Make attempt and outcome recording durable, and extend the policy model
   separately for messaging delivery and provider capabilities.

## Number provisioning: implemented

BYOC numbers are created against customer-authorized carrier connections.
Managed purchasing persists a durable order before DIDWW submission, records
uncertain provider outcomes, and reconciles by provider order reference.
The managed DID is activated after provider ownership and its configured
inbound delivery trunk have been independently verified. An accepted purchase
request alone does not mean the number is active.

## Number lifecycle: implemented and limitations

The `number_lifecycle_operations` table keeps operation identity,
idempotency keys, state, and reconciliation schedule separate from the
resulting phone number. Its operation enum admits assignment and port-out,
but this does not mean full assignment or port-out workflows are exposed.

Managed release disables the number before requesting DIDWW termination.
The operation is marked submitted in the same local transaction that disables
the number, before the external request starts. A worker verifies the DIDWW
terminated state instead of blindly repeating a release after a restart or
ambiguous response. If the provider request did not execute, release can
remain pending external confirmation and requires an explicit recovery
procedure; automatic resubmission is not permitted without proven safety.
Local completion and its outbox event are committed together.

Emergency registrations are versioned. The current implementation deactivates
the prior current record and creates a new pending version before calling
provider validation. It therefore does **not** guarantee continuity of the
previously active mapping if replacement validation fails. Replacement
sequencing and provider-specific regulatory requirements must be addressed
before treating this flow as production-ready. The default DIDWW adapter
returns a capability-unavailable error for emergency validation.

Port-in cases store tenant-scoped document metadata, including object keys and
SHA-256 strings, but the current API does not verify object existence,
ownership, checksum contents, or provider access. These checks must precede
a production provider submission.

Port-in reconciliation currently submits when documents are recorded and no
provider reference is stored. If the submission response is ambiguous, a
later reconciliation can submit again. A durable submission claim, provider
reference recovery, and manual-review state must be implemented before
real port-ins are enabled. The default DIDWW adapter does not submit or query
port-ins, and explicitly returns a capability-unavailable error.

When a provider confirms activation, the local port-in can advance directly
from submitted or in-progress to activated even if the intermediate
firm-order-commit update was missed. A firm-order-commit timestamp must never
be fabricated from a later activation callback. The local number, port-in case,
lifecycle completion, and outbox event are committed together.

## Next lifecycle integration slices

1. Add safe, single-submission claims and provider reference recovery for
   port-ins, including concurrent-worker and timeout tests.
2. Design emergency-address replacement that preserves a valid existing
   mapping wherever provider capabilities and applicable requirements allow.
3. Implement document object authorization and checksum verification.
4. Complete provider-specific porting and emergency capabilities only against
   verified provider contracts.
5. Add explicit manual-review and operator-recovery paths for uncertain
   release, port-in, and assignment outcomes.
