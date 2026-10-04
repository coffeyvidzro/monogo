# Leamout architecture

Leamout is a carrier-grade platform for autonomous voice agents that can be
self-hosted or used as a managed cloud service. It connects customer-owned
telephony infrastructure to realtime AI agents while providing call control,
media orchestration, model orchestration, tool execution, interruption
handling, routing, and observability.

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
- support both Self-Hosted and Leamout Cloud deployments without changing the
  execution model;
- keep durable control-plane state separate from active per-session runtime
  state;
- remain modular without prematurely turning every module into a network
  service.

## System boundaries

Leamout has three primary planes.

### 1. Telephony Runtime

The Telephony Runtime owns SIP signaling, database-backed SIP routing, call
execution, RTP anchoring, and integration with the realtime media process.

```text
customer carrier / PBX / SBC
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
- authentication and routing using customer-owned trunk configuration;
- call establishment and teardown;
- answer, bridge, transfer, hold, playback, DTMF, and recording;
- codec and media negotiation;
- RTP anchoring and media policy;
- WebRTC connectivity when required.

Component roles:

- **OpenSIPS** performs SIP routing, authentication, and policy enforcement
  against database-backed customer trunk configuration.
- **FreeSWITCH** is the B2BUA, call-application, and media-control runtime.
- **RTPengine** is the RTP anchoring and media boundary.
- **Coturn** provides STUN/TURN for WebRTC scenarios that require ICE relay.

OpenSIPS is not a Leamout-managed carrier or shared SIP network. It is part of
the telephony runtime of the deployment in which Leamout is running.

Coturn is not a required hop for normal SIP trunk traffic.

#### SIP trunk model

Leamout models the technical SIP connection rather than the commercial carrier
relationship. A trunk is the tenant-owned connectivity root and contains its
authentication policy, source CIDRs, codecs, admission limits, and one or more
remote SIP endpoints.

```text
SIP Trunk
   |
   +-- authentication / source CIDRs
   +-- codecs / CPS / concurrency
   |
   +-- remote endpoint 1
   +-- remote endpoint 2
   +-- ...
```

The remote endpoint is supplied by the customer or the customer's carrier,
PBX, SBC, or SIP platform. Leamout stores and resolves it from PostgreSQL.

For example:

```text
mytrunk.pstn.twilio.com:5061
sip.provider.example:5060
pbx.customer.example:5061
10.20.30.40:5060
```

Provider identity is not a separate runtime domain object.

Phone numbers known to Leamout are routing identities attached directly to an
inbound-capable trunk. Calls are attributed to the selected trunk and, for
outbound calls, the selected remote endpoint.

The routing model is:

```text
Control Plane API
      |
      v
PostgreSQL
  |
  +-- trunks
  +-- trunk_endpoints
  +-- trunk credentials
  +-- source networks
  +-- phone numbers
      |
      v
OpenSIPS validates/routes
      |
      v
FreeSWITCH
      |
      v
customer-provided SIP peer
```

Leamout does not insert a Leamout-owned carrier endpoint between the runtime and
that customer-provided peer.

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
- SIP trunks and remote endpoints;
- routing policy;
- credentials and secrets references;
- public API and authorization;
- organization, user, and deployment administration;
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
optional local detector.

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

Media Runtime nodes register themselves in Redis with short-lived heartbeats.
Each registration advertises the node's internal control URL, audio-fork URL,
capacity, active-session count, and drain state.

Agent Runtime placement follows this sequence:

```text
Voice Agent session
      ↓
read healthy media nodes
      ↓
exclude draining/full nodes
      ↓
choose most free capacity
      ↓
atomic Redis capacity lease
      ↓
session_id → media_node_id ownership
      ↓
POST session to selected node
```

The capacity lease and session ownership both expire. A healthy Media Runtime
refreshes ownership for every locally active session during its heartbeat.
This prevents crashed processes from holding capacity indefinitely.

Before graceful drain, a Media Runtime stops its normal heartbeat and advertises
`draining=true`. New sessions are no longer placed there while existing
sessions are closed. Node registration is removed when shutdown completes.

Redis stores coordination only. Audio frames, provider streams, transcripts,
and other live media payloads never pass through Redis.

### PostgreSQL

Use PostgreSQL for durable product state:

- organizations and users;
- agent configuration;
- SIP trunk configuration and remote endpoints;
- routing policy;
- conversation metadata;
- call records;
- audit state;
- durable configuration and history.

## Deployment model

Leamout has two deployment models:

- **Leamout Self-Hosted**, operated by the customer on their infrastructure;
- **Leamout Cloud**, operated and hosted by us.

Both use the same voice-agent platform and execution architecture. Enterprise
is optional commercial packaging for support, security, governance, deployment
assistance, SLAs, and contractual requirements on either deployment model. It
is not a third architecture or separate control plane.

Deployment mode must not change the BYOC boundary. Customers continue to own
their carrier accounts, SIP peers, phone numbers, and carrier spend.

A hosted deployment may expose its own ingress address as ordinary deployment
infrastructure, but that does not turn Leamout into a carrier or create a
Leamout-managed carrier product.

The deployment-neutral architecture can run in environments such as:

- local development;
- Docker-based self-hosting;
- private VPC deployments;
- Kubernetes deployments;
- Leamout-managed cloud infrastructure.

## Service boundaries

The expected process boundaries are intentionally small:

- `server` for HTTP control-plane APIs;
- `worker` for asynchronous work;
- `media` for the realtime media and agent hot path;
- OpenSIPS for database-backed SIP routing and policy;
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
- a Leamout-owned shared SIP carrier network;
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
