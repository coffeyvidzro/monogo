# AGENTS.md

This file defines repository-wide instructions for coding agents working on Monogo, the core runtime behind Leamout.

## Instruction hierarchy

Apply instructions in this order:

1. the explicit task from the user or maintainer;
2. the nearest `AGENTS.md` to the files being changed;
3. this root `AGENTS.md`;
4. canonical architecture documentation under `docs/`;
5. `CONTRIBUTING.md` and existing repository conventions.

A scoped `AGENTS.md` may add stricter rules for its subtree. It should not silently redefine the product architecture.

## Product definition

Leamout is a carrier-grade runtime and control plane for autonomous voice agents.

Monogo is the open-source core runtime implementation.

Leamout sits between customer-owned telephony infrastructure and realtime AI systems. It establishes, controls, processes, and observes autonomous voice calls while customers retain their own carrier relationships, phone numbers, SIP trunks, PBXs, SBCs, and carrier billing.

Leamout is BYOC-only.

Do not reintroduce any of the following unless the task explicitly changes the product strategy:

- managed-carrier services;
- carrier resale or wholesale telecom billing;
- telecom wallets or per-minute carrier charging;
- number resale or managed number inventory;
- general-purpose SMS, SMPP, or WhatsApp products;
- payment-provider integrations unrelated to the current runtime product.

## Architecture boundaries

Preserve the separation between the Control Plane, Agent Runtime, Telephony Runtime, and realtime Media Runtime.

### Control Plane

The Control Plane owns durable configuration and management, including organizations, identities, agents, SIP trunks, routing policy, provider credentials, APIs, webhooks, diagnostics, and observability.

The Control Plane must not become part of the live audio hot path.

### Agent Runtime

The Agent Runtime owns per-call conversational execution, including:

- session and turn state;
- interruption and barge-in handling;
- model orchestration;
- tool execution;
- playback cancellation;
- human handoff;
- conversation state and runtime diagnostics.

### Telephony Runtime

The Telephony Runtime owns SIP signaling, call execution, and media integration.

The expected infrastructure roles are:

- OpenSIPS: SIP edge, authentication, routing, and policy enforcement;
- FreeSWITCH: B2BUA and call/media control;
- RTPengine: RTP anchoring and media boundary;
- Coturn: STUN/TURN when WebRTC requires it;
- SIP trunks: organization-owned connectivity to carriers, PBXs, SBCs, or other SIP peers.

A carrier or provider identity is not a separate first-class product object merely because a SIP trunk connects to one.

### Media Runtime

Latency-sensitive audio processing belongs in the realtime path.

Do not:

- publish raw audio frames through NATS JetStream;
- persist raw realtime audio frames to PostgreSQL as normal runtime behavior;
- introduce synchronous database work into per-frame audio processing;
- introduce unbounded queues or channels into realtime paths;
- perform avoidable blocking network work inside audio loops.

Prefer bounded buffering, explicit timeouts, cancellation-aware operations, deterministic cleanup, and graceful drain behavior.

## Provider neutrality

External AI providers adapt into Leamout contracts; they do not define Leamout domain models.

Deepgram, Groq, Cartesia, OpenAI, and future providers should remain behind provider-neutral STT, LLM, TTS, or realtime contracts.

Do not leak vendor-specific request or response structures into core runtime models unless the task explicitly requires a provider-specific surface.

Provider credentials must remain tenant-scoped and must not become process-global configuration.

## Security baseline

Treat tenant isolation and credential handling as hard requirements.

Never:

- remove organization scoping to simplify a query or test;
- log API keys, SIP passwords, access tokens, private keys, or decrypted provider secrets;
- persist plaintext provider credentials;
- place secrets in engine snapshots, events, fixtures, screenshots, or diagnostics;
- disable TLS verification to make an integration work;
- weaken authorization or replay protections to make tests pass;
- expose internal privileged operations as public HTTP endpoints without an explicit product requirement.

Security-sensitive changes must consider authorization, tenant boundaries, credential scope, replay resistance, tool execution, SIP abuse, media-session isolation, failure handling, and secret redaction.

Follow `SECURITY.md` for vulnerability reporting.

## Scope discipline

Keep changes focused on the requested task.

Do not opportunistically redesign unrelated modules, rename broad areas of the repository, rewrite migrations, or perform formatting sweeps unless those changes are required for correctness.

When a task uncovers unrelated problems, leave them unchanged unless they block the requested work. Mention them separately when useful.

Prefer small, reviewable changes over speculative abstractions.

## Repository conventions

Respect existing module boundaries and naming.

Before introducing a new package, service, process, or cross-module abstraction, verify that an existing domain module cannot own the behavior cleanly.

Do not split a product module into a network service merely for code organization. A network boundary should be justified by scaling, isolation, latency, security, or deployment requirements.

Do not manually edit generated code when a generator is the source of truth.

Keep comments focused on why behavior exists, not on restating obvious code.

## Database changes

Use migrations and sqlc where the server already relies on them.

The expected flow is:

```text
migration
  -> sqlc query
  -> sqlc generate
  -> repository
  -> service/runtime
  -> tests
```

Do not scatter new raw SQL through services when the query belongs in sqlc.

Preserve organization scoping for tenant-owned rows.

Keep migrations focused and forward-moving. Avoid unrelated schema cleanup in the same change.

## Validation

Run the checks relevant to the files you changed.

For server work, the baseline is:

```sh
cd server
go test ./...
go vet ./...
go build ./...
```

For concurrency-sensitive runtime work, also run:

```sh
go test -race ./...
```

For client work, follow `clients/AGENTS.md`.

For deployment changes, validate the Compose configuration when relevant:

```sh
docker compose --env-file .env.example -f deploy/compose.yaml config --quiet
```

Do not claim a check passed unless it was actually run or CI confirms it.

## Documentation

Update documentation when a change modifies public APIs, architecture, provider contracts, deployment requirements, security assumptions, SIP behavior, media behavior, or operational procedures.

Architecture documentation under `docs/` should describe durable system decisions. Avoid documenting temporary implementation accidents as architecture.

## Completion standard

Before considering a task complete:

- verify the change matches the requested scope;
- confirm architecture boundaries remain intact;
- confirm tenant and credential isolation are preserved;
- add or update tests for changed behavior when practical;
- run relevant formatting, lint, test, and build checks;
- update documentation when the public or architectural contract changed;
- avoid leaving dead code, stale configuration, or obsolete generated output behind.
