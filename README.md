# Monogo

**Leamout is a carrier-grade runtime and control plane for autonomous voice agents. It connects customer-owned telephony infrastructure to realtime AI agents while providing call control, media orchestration, model orchestration, tool execution, interruption handling, routing, observability, and enterprise deployment infrastructure.**

Monogo is the core runtime implementation behind Leamout.

Leamout sits between customer-owned telephony infrastructure and realtime AI
systems. It owns the runtime required to establish, control, process, and
observe autonomous voice calls while customers keep their carrier relationships,
phone numbers, SIP trunks, PBXs, and carrier billing.

Leamout is not a general-purpose CPaaS, telecom reseller, or managed carrier.

## Architecture

Leamout is organized around three explicit runtime boundaries:

```text
                              Leamout

+------------------------------------------------------------------+
|                         CONTROL PLANE                            |
|                                                                  |
|  Auth  Tenancy  Agents  Routing  Credentials  API  Webhooks      |
|  Observability  Runtime registration  Deployment configuration   |
+-------------------------------+----------------------------------+
                                |
                                | configuration and control
                                v
+------------------------------------------------------------------+
|                         AGENT RUNTIME                            |
|                                                                  |
|  Agent sessions  Turn control  Barge-in  Model orchestration     |
|  Tool execution  Conversation state  Human handoff               |
+-------------------------------+----------------------------------+
                                |
                                | realtime audio
                                v
+------------------------------------------------------------------+
|                       TELEPHONY RUNTIME                          |
|                                                                  |
|  OpenSIPS        FreeSWITCH        RTPengine        Coturn        |
|  SIP edge        Call execution    Media boundary   WebRTC TURN   |
+-------------------------------+----------------------------------+
                                |
                                v
                       customer-owned
                         connectivity

                 SIP carrier · PBX · SBC · WebRTC
```

The control plane configures the runtime but does not sit in the live audio hot
path. The Agent Runtime owns per-call conversational execution. The Telephony
Runtime owns SIP signaling, call control, and media integration.

See [docs/architecture.md](docs/architecture.md) for the canonical system
architecture and [docs/media-plane.md](docs/media-plane.md) for realtime media
contracts and runtime status.

## Product boundary

Leamout provides:

- carrier-grade SIP and media infrastructure;
- programmable call control;
- realtime autonomous voice-agent sessions;
- turn detection and interruption handling;
- composable STT, LLM, and TTS orchestration;
- integrated realtime model support;
- enterprise tool execution;
- routing and human handoff;
- recordings, events, webhooks, and observability;
- self-hosted, private-cloud, and cloud deployment infrastructure.

Leamout does not buy or resell carrier minutes, phone numbers, SMS, WhatsApp
capacity, or managed carrier services.

## BYOC only

SIP connectivity is organization-owned.

A customer connects an existing SIP carrier, PBX, SBC, or supported WebRTC
endpoint through a Leamout SIP trunk and continues using the external accounts
and phone numbers they already control.

```text
Customer carrier / PBX / SBC
             |
             v
       Leamout SIP trunk
             |
             v
          OpenSIPS
             |
             v
        FreeSWITCH
             |
      RTPengine media
             |
             v
   Leamout Media Runtime
             |
             v
    Leamout Agent Runtime
```

A SIP trunk is Leamout's technical connectivity primitive. The peer on the
other side may be a carrier, PBX, SBC, or another SIP platform; provider
identity is not a separate runtime domain object.

## Agent runtime

The Agent Runtime owns the state and execution of a live autonomous voice
session:

- conversation and turn state;
- interruption and barge-in handling;
- streaming model orchestration;
- tool-call authorization and execution;
- playback cancellation;
- human handoff;
- runtime events and diagnostics.

Latency-sensitive session state stays local to the active runtime whenever
possible. Redis is used for distributed coordination where cross-process state
is actually required.

The media runtime supports two engine models:

- **Composable**: streaming STT, LLM, and TTS providers coordinated by Leamout.
- **Integrated**: an end-to-end realtime voice model.

Current provider integrations include Deepgram, Groq, Cartesia, and OpenAI
Realtime. Provider packages adapt external protocols into Leamout's
provider-neutral media contracts; providers do not define the product
architecture.

## Media path

Live audio stays on the realtime media path.

Audio frames are not published through NATS JetStream and are not persisted to
PostgreSQL.

```text
Caller RTP
    |
    v
RTPengine
    |
    v
FreeSWITCH
    |
    | bidirectional audio stream
    v
Go Media Runtime
    |
    v
Agent Runtime
    |
    +--> composable STT -> LLM -> TTS
    |
    +--> integrated realtime model
```

Immediate actions such as interruption, playback cancellation, and turn
transitions occur inside the runtime. Durable events can be emitted
asynchronously after the runtime has acted.

## Telephony runtime

The telephony layer remains a first-class part of the product.

- **OpenSIPS**: SIP edge, authentication, routing, and policy enforcement.
- **FreeSWITCH**: B2BUA, call application, and media-control runtime.
- **RTPengine**: RTP anchoring and media boundary.
- **Coturn**: STUN/TURN services when WebRTC connectivity requires them.
- **SIP trunks**: organization-owned authentication, source CIDRs, codecs,
  admission limits, and gateway endpoints.
- **Phone numbers**: customer-owned voice bindings attached directly to trunks.
- **Calls and recordings**: programmable voice primitives used by
  the Agent Runtime.

## Control plane

The control plane owns durable configuration and management:

- organizations and identities;
- agents and agent configuration;
- SIP trunks and routing policy;
- model and tool credentials;
- API and authorization;
- runtime registration and deployment configuration;
- events, webhooks, diagnostics, and observability.

Its infrastructure includes:

- **PostgreSQL** for durable product and control-plane state;
- **Redis** for admission, coordination, ephemeral distributed state, and rate
  limiting;
- **NATS JetStream** for durable asynchronous events and background work.

NATS is not part of the live audio path.

## Tool execution

Models do not receive unrestricted database or infrastructure access.

Agent tool calls flow through Leamout's tool runtime, which is responsible for
authorization, schema validation, timeouts, cancellation, auditing, and
execution against approved customer systems.

```text
Model
  |
  v
Tool request
  |
  v
Leamout Tool Runtime
  |
  +-- authorize
  +-- validate
  +-- execute
  +-- audit
  |
  v
Customer API / CRM / service
```

## Commercial boundary

Billing and payment processing are intentionally outside the current runtime
architecture while the commercial model is redesigned separately.

Customer carrier spend remains outside Leamout.

## Services

- `server`: HTTP control plane and public API.
- `worker`: asynchronous jobs, event consumers, and background coordination.
- `media`: low-latency realtime audio and AI-provider orchestration.
- `opensips`: public SIP edge and routing.
- `freeswitch`: call application and media runtime.
- `rtpengine`: RTP/media boundary.
- `coturn`: WebRTC STUN/TURN when required.

The Go codebase remains modular. Product modules should not become independent
network services unless scaling, isolation, or deployment requirements justify
that boundary.

## Repository direction

Current engineering work is focused on turning the existing telephony platform
into a unified autonomous voice-agent runtime:

```text
telephony runtime
      |
      v
realtime media
      |
      v
agent sessions
      |
      +-- turn control
      +-- barge-in
      +-- model orchestration
      +-- tool execution
      +-- human handoff
      +-- observability
```

General-purpose messaging, managed carrier commerce, telecom resale, telecom
wallet charging, and payment adapters are outside the current product boundary.

## Validation

Run the server checks:

```sh
cd server

go test ./...
go vet ./...
go build ./...
```

Validate the deployment configuration:

```sh
cd ..

docker compose --env-file .env.example -f deploy/compose.yaml config --quiet
```

Copy `.env.example` to a protected environment file and replace every example
secret before starting a deployment.
