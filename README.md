# Monogo

**Leamout is the carrier-grade runtime and control plane for autonomous voice agents.**

Monogo is the core runtime implementation behind Leamout. It connects
customer-owned SIP carriers, trunks, phone numbers, PBXs, and WebRTC endpoints
to realtime autonomous voice agents.

Leamout owns call control, media orchestration, agent execution, routing,
observability, and platform operations. Customers keep their carrier
relationships and pay their carriers directly.

Leamout is not a general-purpose CPaaS or telecom reseller.

## Product model

```text
                       Leamout / Monogo

                    Agent Control Plane
                            |
                    Realtime Agent Runtime
                            |
                     Telephony Runtime
                            |
              +-------------+-------------+
              |             |             |
           SIP trunk       PBX          WebRTC
              |             |             |
              +-------------+-------------+
                            |
                    customer-owned
                       connectivity
```

The core product boundary is:

- carrier-grade SIP and media infrastructure;
- programmable call control;
- realtime AI voice sessions;
- turn detection and interruption handling;
- STT, LLM, and TTS provider orchestration;
- enterprise tool execution and handoff;
- recordings, events, webhooks, and observability;
- self-hosted and cloud deployment.

Customers bring their own carrier connectivity. Leamout does not buy or resell
carrier minutes, phone numbers, SMS, or WhatsApp capacity.

## BYOC only

Carrier connectivity is organization-owned.

A customer can connect an existing SIP carrier, trunk, PBX, or SBC and continue
using the carrier account and phone numbers they already control.

Leamout provides the control plane around that connectivity:

```text
Customer carrier / PBX
         |
         v
      OpenSIPS
         |
   RTPengine / FreeSWITCH
         |
         v
   Leamout Agent Runtime
         |
   +-----+------+------+
   |            |      |
  STT          LLM    TTS
```

Carrier providers are descriptive catalog metadata only. Monogo does not ship
provider-owned DIDWW, CommPeak, SMPP, WhatsApp, or other carrier-specific
commerce adapters.

## AI voice runtime

Live audio stays on the realtime media path. Audio frames are not published
through NATS or persisted to PostgreSQL.

The media runtime supports two engine models:

- **Composable**: streaming STT, LLM, and TTS providers coordinated by Leamout.
- **Integrated**: an end-to-end realtime voice model.

Current provider integrations include Deepgram, Groq, Cartesia, and OpenAI
Realtime. Provider integrations translate external protocols into the
provider-neutral media session model rather than defining the product
architecture.

See [docs/media-plane.md](docs/media-plane.md) for the media contracts,
transport model, and runtime status.

## Telephony runtime

The telephony layer remains a first-class part of the product.

- **OpenSIPS**: SIP edge, authentication, and routing.
- **FreeSWITCH**: call application and media control runtime.
- **RTPengine**: RTP anchoring and media boundary.
- **Coturn**: TURN services for WebRTC.
- **Carrier connections**: customer-owned SIP authentication and source-IP
  configuration.
- **Trunks and routing**: organization-owned call paths and endpoint health.
- **Phone numbers**: customer-owned voice bindings.
- **Calls, conferences, and recordings**: programmable voice primitives used by
  the Agent Runtime.

## Commercial boundary

Billing and payment processing are intentionally not part of the current
runtime. The repository is focused on telephony, realtime media, and autonomous
voice-agent infrastructure while the commercial model is redesigned separately.

Customer carrier spend remains outside Leamout.


## Services

- `server`: HTTP control plane and public API.
- `worker`: asynchronous jobs, event consumers, reconciliation, and SIP
  endpoint health checks.
- `media`: low-latency realtime audio and AI provider orchestration.
- `opensips`: public SIP edge and routing.
- `freeswitch`: call application and media runtime.
- `rtpengine`: RTP/media boundary.
- PostgreSQL: durable product and control-plane state.
- Redis: realtime coordination, admission state, and ephemeral state.
- NATS JetStream: durable asynchronous events. It is not part of the live audio
  path.

## Repository direction

Current engineering work is focused on turning the existing telephony platform
into a unified Agent Runtime:

```text
telephony core
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

General-purpose messaging, managed carrier commerce, and telecom resale are
outside the current product boundary.

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
