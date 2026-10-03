# Voice Agent execution plan

This document turns the remaining Voice Agent work into an ordered delivery
plan. The goal is not to add more architectural layers. It is to complete and
prove the existing control-plane, Agent Runtime, Media Runtime, and telephony
boundaries as one product.

## Product target

A production-ready Voice Agent must be able to receive or place a call, attach
an immutable agent configuration, stream audio through the selected AI engine,
execute authorized tools, optionally record the call, persist conversation
history, and terminate cleanly. Every step must be organization-scoped,
observable, and recoverable.

The initial product remains bring-your-own-carrier and bring-your-own-AI. A
future Leamout-managed AI offering must reuse the same provider and session
contracts rather than introduce a second runtime path.

## Delivery principles

- Keep credentials owned by organizations and encrypted at rest.
- Keep provider secrets out of Voice Agent configuration snapshots, logs,
  events, and API responses.
- Resolve an immutable runtime configuration before live media starts.
- Keep PCM and latency-sensitive control traffic out of PostgreSQL and NATS.
- Reject an unready agent before answering or originating billable media.
- Prefer concrete Go services and structs. Introduce interfaces only at active
  provider or consumer boundaries.
- Make the golden end-to-end call the release unit. A component is not complete
  merely because its unit tests pass.

## Phase 1: organization AI integrations

Promote the existing provider credentials into product-level organization
integrations for Deepgram, Groq, Cartesia, and OpenAI.

An integration should expose:

- stable identity, provider, display name, and organization ownership;
- encrypted secret material with create, rotate, and revoke operations;
- connection state: `unchecked`, `ready`, `invalid`, or `unavailable`;
- last verification time and a sanitized failure code;
- provider capabilities and the Voice Agents currently using it;
- an explicit verification action that performs a minimal provider-specific
  authentication or capability check.

Secret values are write-only. Listing or retrieving integrations must never
return ciphertext, plaintext, or secret-derived debug data. Rotation must not
mutate the immutable configuration of a session that is already active.

### Phase 1 exit criteria

- All four built-in providers support organization-scoped create, list,
  verify, rotate, and revoke workflows.
- Cross-organization reads and bindings fail in repository and service tests.
- Provider errors are normalized into stable onboarding failure codes.
- Audit events cover secret creation, verification, rotation, and revocation.
- Deployment-global AI keys are development fallbacks only and cannot silently
  override an organization binding.

## Phase 2: Voice Agent configuration and presets

Expose two engine modes in the product API:

1. **Composable**: one `stt`, one `llm`, and one `tts` integration.
2. **Realtime**: one integrated `realtime` integration.

Voice Agent configuration should remain provider-neutral at the top level:
instructions, language, voice, tools, interruption policy, recording policy,
and engine mode. Provider-specific options belong to the relevant provider
binding and are validated by that provider adapter.

### Presets

Presets are versioned templates, not new runtime engines:

| Preset | Initial topology | Product intent |
| --- | --- | --- |
| Balanced | Deepgram + Groq + Cartesia | Default quality, latency, and cost balance |
| Fast | Deepgram + Groq + Cartesia with latency-first options | Lowest practical response latency |
| High Intelligence | Deepgram + a higher-capability LLM + Cartesia | Reasoning quality over minimum latency |
| Realtime | OpenAI Realtime | Integrated speech-to-speech interaction |

Applying a preset copies its versioned values into an editable Voice Agent
draft. Existing agents do not change when a preset is updated. The activated
agent revision, not the mutable preset, is snapshotted into every live session.

### Phase 2 exit criteria

- Create and update APIs accept either a preset or explicit engine bindings.
- Provider configuration is schema-validated before activation.
- Agents have draft and active revisions, or an equivalent immutable
  activation model.
- The API returns a sanitized, actionable readiness report.
- Preset versions and the resolved configuration are visible in diagnostics.

## Phase 3: readiness and activation gates

Readiness is a first-class service result, not a UI-only checklist. It should
evaluate the complete configuration required to start a call.

For a composable agent, readiness requires:

- active STT, LLM, and TTS bindings;
- ready organization integrations for each binding;
- provider-compatible model, language, voice, and audio configuration;
- valid tool schemas and reachable tool configuration where verification is
  supported;
- valid telephony routing and capacity.

For a realtime agent, the STT, LLM, and TTS requirements are replaced by one
ready realtime binding.

Readiness must return stable machine codes, field paths, and remediation text.
The same readiness evaluator must be used by activation, inbound attachment,
outbound origination, the public API, and the product UI. Runtime attachment
still performs a final check to protect against credentials revoked after
activation.

### Phase 3 exit criteria

- An unready agent cannot be activated or attached to a call.
- Readiness changes when a credential is revoked, a required binding is
  removed, or an incompatible option is saved.
- Concurrent updates cannot activate a stale configuration revision.
- Readiness checks do not reveal provider secrets.

## Phase 4: golden call lifecycle

Harden one complete path before broadening the product surface:

```text
SIP trunk
  -> routed call
  -> Voice Agent revision
  -> Media Runtime placement
  -> AI provider session
  -> tool execution
  -> optional recording
  -> conversation turns and summary
  -> terminal call/session state
```

The lifecycle must be idempotent across duplicate and out-of-order telephony
events. Cleanup must cover partial failure after every acquired resource:
durable session, media-node lease, control WebSocket, audio fork, provider
session, tool execution, and recording.

Required reconciliation includes worker restart, Media Runtime loss, control
channel loss, provider disconnect, caller hangup during a tool call, recording
finalization failure, and durable sessions left active without a live channel.

### Phase 4 exit criteria

- One inbound and one outbound golden call pass against real FreeSWITCH and at
  least one real AI provider.
- Barge-in cancels stale provider output and clears buffered playback.
- User, assistant, and tool turns are stored in deterministic order.
- Recording and conversation records share call and session correlation IDs.
- All terminal paths release ownership and reach exactly one terminal state.
- Reconciliation closes orphaned calls and sessions after process restarts.

## Phase 5: carrier-grade operations and release gates

### Observability

Correlate organization, call, channel, agent revision, Voice Agent session,
media node, provider session, tool execution, and recording identifiers. Add
metrics and traces for:

- answer-to-media-attachment latency;
- end-of-speech to first output audio;
- input and output queue pressure;
- interruption count and cancellation latency;
- provider connection, timeout, and terminal failure rates;
- tool duration and outcome;
- recording finalization;
- orphan reconciliation and forced cleanup.

Logs and traces must redact provider secrets, signed media URLs, tool
credentials, and sensitive transcript content according to retention policy.

### Release gates

A release is blocked unless all of the following pass:

- unit and race tests for the changed Go packages;
- provider contract and adapter conformance tests;
- tenant-isolation and credential-redaction tests;
- deterministic Media Runtime integration tests with fault injection;
- real-stack inbound and outbound acceptance calls;
- barge-in, tool, recording, and conversation-history scenarios;
- restart, drain, orphan-reconciliation, and dependency-failure scenarios;
- concurrency load and long-running soak tests with latency thresholds;
- migration upgrade and rollback validation.

## Phase 6: product API and UI

Build the product experience on the same application services used by runtime
gates. The initial product surface should include:

- organization AI integration onboarding and verification;
- Voice Agent creation from a preset or explicit configuration;
- readiness checklist and activation;
- phone-number and route binding;
- test-call launch and live diagnostics;
- calls, recordings, transcripts, tool executions, and failure timelines;
- credential rotation impact and integration usage views.

The UI must not duplicate readiness rules or infer runtime support from provider
names. It renders capabilities and remediation returned by the API.

## Later: managed AI and commercial accounting

Leamout-managed AI is a later credential-sourcing policy, not a new engine.
When introduced, the runtime should receive the same provider-neutral binding
and ephemeral runtime secret shape used for customer-owned integrations.

Usage accounting should consume normalized provider usage and call lifecycle
events asynchronously. Billing, quotas, or pricing must never enter the live
audio path. Before monetization, usage records need idempotency, reconciliation
against provider invoices, tenant attribution, and explicit handling for
provider sessions that fail before reporting final usage.

## Recommended execution slices

Deliver the plan as vertical slices:

1. Deepgram organization integration plus verification and readiness output.
2. Groq and Cartesia integrations, completing the composable readiness gate.
3. OpenAI integration and the realtime readiness gate.
4. Versioned presets and immutable agent activation.
5. One inbound golden call with tools, recording, and conversation history.
6. Outbound golden call and lifecycle reconciliation.
7. Operational dashboards, fault-injection tests, load tests, and release gate.
8. Product-facing onboarding, agent builder, test call, and call history.

Each slice includes its API, persistence, runtime wiring, tests, observability,
and minimal product UI. Avoid separate “backend complete” and “UI complete”
milestones that allow the two surfaces to drift.
