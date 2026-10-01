# Leamout architecture

Leamout is a carrier-grade runtime and control plane for autonomous voice
agents. It connects customer-owned telephony infrastructure to realtime AI
agents while providing call control, media orchestration, model orchestration,
tool execution, interruption handling, routing, observability, and enterprise
deployment infrastructure.

This document defines the product and system boundaries that guide Monogo
development.

## Design goals

Leamout should:

- keep carrier connectivity customer-owned;
- preserve carrier-grade SIP and media primitives;
- keep latency-sensitive audio and interruption handling off asynchronous buses;
- keep provider integrations behind provider-neutral contracts;
- support both composable and integrated realtime AI engines;
- make tool execution explicit, authorized, observable, and cancellable;
- allow self-hosted and private-cloud deployment without changing the product
  model;
- keep durable control-plane state separate from active per-session runtime
  state;
- remain modular without prematurely turning every module into a network
  service.

## System boundaries

Leamout has three primary planes.

### 1. Telephony Runtime

The Telephony Runtime owns SIP signaling, call execution, RTP anchoring, and
integration with the realtime media process.

```text
Customer carrier / PBX / SBC
             |
             | SIP
             v
          OpenSIPS
             |
             | SIP
             v
        FreeSWITCH
             |
        RTPengine
             |
             | bidirectional audio
             v
       Media Runtime
```

Responsibilities:

- SIP ingress and egress;
- authentication and routing;
- call establishment and teardown;
- answer, bridge, transfer, hold, playback, DTMF, and recording;
- codec and media negotiation;
- RTP anchoring and media policy;
- WebRTC connectivity when required.

Component roles:

- **OpenSIPS** is the SIP edge and routing layer.
- **FreeSWITCH** is the B2BUA, call-application, and media-control runtime.
- **RTPengine** is the RTP anchoring and media boundary.
- **Coturn** provides STUN/TURN for WebRTC scenarios that require ICE relay.

Coturn is not a required hop for normal SIP trunk traffic.

### 2. Agent Runtime

The Agent Runtime owns the realtime execution of an autonomous voice session.

```text
                  AgentSession
                       |
         +-------------+-------------+
         |             |             |
     turn state    tool runtime   handoff
         |
         +---------------------------+
         |                           |
    Composable                   Integrated
         |                           |
   STT -> LLM -> TTS          realtime model
```

Responsibilities:

- session lifecycle;
- live conversation state;
- turn detection;
- interruption and barge-in;
- model orchestration;
- output cancellation and playback control;
- tool execution;
- human handoff;
- normalized runtime events and diagnostics.

The runtime owns immediate decisions. It must not require NATS round trips for
actions such as stopping playback when a caller interrupts.

### 3. Control Plane

The Control Plane owns configuration, policy, administration, and durable
state.

Responsibilities:

- identity and tenancy;
- agents and agent configuration;
- telephony connections;
- routing policy;
- credentials and secrets references;
- public API and authorization;
- runtime registration and deployment configuration;
- webhooks and event subscriptions;
- observability and audit state.

Core infrastructure:

- **PostgreSQL** stores durable relational state.
- **Redis** supports distributed admission, coordination, rate limiting, and
  other short-lived cross-process state.
- **NATS JetStream** carries durable asynchronous events and background work.

These infrastructure components support the control plane; they are not the
control plane itself.

## Realtime media path

Live media follows a direct, bounded path:

```text
Caller RTP
    |
    v
RTPengine
    |
    v
FreeSWITCH
    |
    | PCM / negotiated audio stream
    v
Go Media Runtime
    |
    v
Agent Runtime
    |
    +--> STT -> LLM -> TTS
    |
    +--> integrated realtime engine
```

Architecture rules:

1. Audio frames do not traverse NATS JetStream.
2. Audio frames are not persisted to PostgreSQL.
3. Per-session latency-sensitive state stays local to the active runtime when
   possible.
4. Provider WebSocket streams belong to one media session and are not reused
   across unrelated calls.
5. Provider-specific payloads are normalized at adapter boundaries.
6. Immediate interruption behavior happens inside the runtime before telemetry
   is emitted asynchronously.

## Engine model

Leamout supports two provider models.

### Composable

A composable engine combines independent streaming providers:

```text
audio -> speech recognition -> language model -> speech synthesis -> audio
```

Turn detection can be supplied by the speech-recognition provider or by an
optional local detector such as Silero.

### Integrated

An integrated engine uses one realtime provider that owns speech input,
generation, and speech output behind a single session.

Integrated and composable engines are alternatives. An integrated engine is not
an additional stage inside the composable pipeline.

## Provider boundary

External AI providers are adapters, not architecture.

Provider packages translate remote APIs and protocols into Leamout's normalized
session contracts. The runtime should make decisions based on those contracts,
not on provider-specific payload shapes.

The architecture should remain stable when providers are added, replaced, or
removed.

## Tool runtime

Tool execution is part of the Agent Runtime but has a security boundary of its
own.

```text
LLM / realtime model
         |
         v
   normalized tool call
         |
         v
   Leamout Tool Runtime
         |
         +-- authorization
         +-- schema validation
         +-- timeout
         +-- cancellation
         +-- audit
         +-- execution policy
         |
         v
Customer-controlled API / CRM / service
```

Models must not receive arbitrary database or infrastructure access.

Tool results are returned to the active agent session using the normalized
runtime contract.

## Event model

Events describe runtime transitions and make them observable to the rest of the
platform.

Examples include:

- call started;
- call answered;
- session started;
- caller speech started;
- caller speech ended;
- agent response started;
- agent interrupted;
- tool call requested;
- tool call completed;
- handoff started;
- call ended.

The runtime acts first for latency-sensitive operations, then publishes durable
events asynchronously where appropriate.

## State model

Use the narrowest state scope possible.

### Process-local session state

Prefer local state for:

- active turn state;
- partial transcripts;
- model-generation state;
- active tool calls;
- playback state;
- cancellation state;
- interruption state.

### Redis

Use Redis for state that must cross process or node boundaries:

- admission and concurrency counters;
- session ownership or node lookup;
- distributed rate limits;
- short-lived coordination;
- locks only where a distributed lock is actually necessary.

### PostgreSQL

Use PostgreSQL for durable product state:

- organizations and users;
- agent configuration;
- telephony configuration;
- routing policy;
- conversation metadata;
- call records;
- audit state;
- durable configuration and history.

## Deployment model

The product architecture is deployment-neutral.

The same runtime model should support:

- local development;
- Docker-based self-hosting;
- private VPC deployments;
- Kubernetes deployments;
- Leamout-managed cloud infrastructure.

Deployment mode must not change the BYOC boundary: customer carrier
relationships and carrier spend remain customer-owned.

## Service boundaries

The expected process boundaries are intentionally small:

- `server` for HTTP control-plane APIs;
- `worker` for asynchronous work;
- `media` for the realtime media and agent hot path;
- OpenSIPS;
- FreeSWITCH;
- RTPengine;
- Coturn when required;
- PostgreSQL;
- Redis;
- NATS JetStream.

Go domain modules should remain in-process unless independent scaling, failure
isolation, security, or deployment requirements justify a separate service.

## Product exclusions

The current Leamout architecture does not include:

- managed carrier resale;
- Leamout-owned carrier minutes;
- retail or wholesale telecom rating;
- prepaid telecom wallets;
- carrier payment settlement;
- SMS or WhatsApp CPaaS;
- generic messaging infrastructure;
- payment adapters as part of the voice runtime.

Those exclusions keep engineering focused on the autonomous voice-agent runtime
and control plane.

## Repository mapping

The existing Monogo structure already maps to these boundaries:

```text
server/internal/

ai/
  agents/
  conversations/
  orchestration/
  tools/

media/
  engine/
  session/
  transport/
  vad/

runtime/
  calling/
  media/
  agent/
  server/
  worker/

telephony/
  calls/
  routing/
  trunks/
  ...
```

Future refactors should be evaluated against this architecture using four
questions:

1. Does the package belong to the Telephony Runtime, Agent Runtime, or Control
   Plane?
2. Is it on the realtime hot path or asynchronous?
3. Is it product logic or a provider/infrastructure adapter?
4. Should it be kept, renamed or moved, merged, or deleted?

The architecture should become simpler as the CPaaS-era responsibilities are
removed.
