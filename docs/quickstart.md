# Leamout quickstart

This guide covers the shortest path from a clean checkout to a working Leamout
Voice Agent.

Leamout is bring-your-own-carrier and bring-your-own-AI. It does not provide a
managed carrier or a Leamout-owned SIP network. The customer provides the SIP
peer and Leamout stores that connectivity as database-backed trunk
configuration.

A real deployment therefore needs:

- a customer-owned SIP carrier, PBX, SBC, or other SIP peer;
- a host for the Leamout runtime;
- credentials for a supported AI provider.

For the first real call, use the integrated OpenAI Realtime path. Once that path
is healthy, you can add the composable Deepgram + Groq + Cartesia pipeline.

## Connectivity model

The SIP peer belongs to the customer.

For example, a customer using a carrier may have a remote SIP endpoint such as:

```text
mytrunk.pstn.twilio.com:5061
```

That remote endpoint is stored on the customer's Leamout trunk. Leamout reads
the trunk, endpoints, authentication policy, source networks, codecs, and
limits from the database when routing calls.

```text
PostgreSQL
  |
  +-- trunks
  +-- trunk_endpoints
  +-- trunk credentials
  +-- trunk source IPs
  +-- phone numbers
  |
  v
OpenSIPS routing / policy
  |
  v
FreeSWITCH
  |
  v
Media Runtime
```

There is no `sip.leamout.com` requirement.

For outbound calls, Leamout routes directly to the customer-provided remote SIP
endpoint stored on the trunk.

```text
Voice Agent
    |
    v
FreeSWITCH
    |
    v
OpenSIPS
    |
    v
customer trunk endpoint
    |
    v
customer carrier / PBX / SBC
```

For inbound calls, the customer's carrier must still be able to deliver SIP to
the customer's Leamout deployment. Depending on the carrier, that destination
may be the deployment's public IP or a customer-controlled hostname. It is
deployment infrastructure, not a Leamout-managed SIP service.

## 1. Prove the runtime locally first

Before connecting real infrastructure, run the repository's mandatory
single-node Voice Agent release gate.

Requirements:

- Docker with Docker Compose;
- Python 3;
- OpenSSL;
- a Linux environment capable of running the telephony containers.

From the repository root:

```sh
sh tests/voice-agent-v1/run.sh
```

The suite starts a disposable stack containing PostgreSQL, Redis, NATS, the API,
worker, Media Runtime, OpenSIPS, FreeSWITCH, RTPengine, a synthetic SIP carrier,
and a local OpenAI Realtime-compatible fixture.

It proves bidirectional audio, Voice Agent session creation, immutable
configuration snapshots, tool execution, conversation persistence, and clean
call termination without requiring a live carrier or AI provider.

To preserve the disposable stack after a failure:

```sh
VOICE_AGENT_V1_KEEP_STACK=1 sh tests/voice-agent-v1/run.sh
```

## 2. Prepare a single-node host

Use a Linux host with a public IPv4 address for the first real deployment.
Avoid Kubernetes and multi-node placement until one inbound and one outbound
call work reliably on a single node.

A practical starting point is:

```text
Ubuntu 24.04 LTS
4 vCPU
8 GB RAM
80 GB or more SSD
public IPv4
Docker Engine
Docker Compose
```

Clone the repository:

```sh
git clone https://github.com/coffeyvidzro/monogo.git
cd monogo
```

## 3. Configure the deployment domain

The HTTP API and recording service need customer-controlled DNS names. TURN
also needs a hostname when WebRTC is used.

For a deployment using `voice.example.com`, a practical layout is:

```text
api.voice.example.com
recordings.voice.example.com
turn.voice.example.com
```

The SIP carrier does not need a Leamout-owned hostname. If the carrier requires
an FQDN for inbound origination, use a hostname controlled by the customer,
such as:

```text
telephony.voice.example.com
```

Otherwise the carrier may be able to target the deployment's public IP
directly. Follow the carrier's SIP origination requirements.

## 4. Prepare the environment

Copy the example environment file:

```sh
cp .env.example .env
```

Replace every example secret before starting a real deployment.

The main deployment values are:

```text
DOMAIN
PUBLIC_IP
CORS_ORIGINS
POSTGRES_PASSWORD
FREESWITCH_ESL_PASSWORD
MEDIA_TOKEN_SECRET
MEDIA_CONTROL_TOKEN
ENCRYPTION_KEY
TURN_AUTH_SECRET
MINIO_ROOT_USER
MINIO_ROOT_PASSWORD
MINIO_APP_ACCESS_KEY
MINIO_APP_SECRET_KEY
```

Generate independent random passwords and tokens. For example:

```sh
openssl rand -hex 32
```

`ENCRYPTION_KEY` must be a raw URL-safe base64 encoding of a 16, 24, or 32 byte
AES key. A 32-byte key can be generated with:

```sh
openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'
```

Do not commit `.env`.

AI provider credentials do not belong in `.env`. They are organization-scoped
secrets stored through the AI integration API and encrypted at rest.

## 5. Provision certificates where required

The current Compose deployment mounts certificates from:

```text
/etc/leamout/certs/fullchain.pem
/etc/leamout/certs/privkey.pem
```

Coturn requires certificates when TURN over TLS is enabled. OpenSIPS also uses
these files when SIP over TLS is enabled for the deployment.

Use certificates for customer-controlled hostnames. Leamout does not require or
provide a shared Leamout SIP hostname.

Certificate bootstrap and renewal are not yet automated by this repository.

## 6. Open only the ports the deployment needs

The default Compose topology publishes:

```text
80/tcp                    HTTP / ACME
443/tcp                   HTTPS
443/udp                   HTTP/3
5060/udp                  SIP
5060/tcp                  SIP
5061/tcp                  SIP over TLS
5062/tcp                  SIP service port
23000-32768/udp           RTPengine media
3478/udp and 3478/tcp     TURN/STUN
5349/udp and 5349/tcp     TURN over TLS
49152-65535/udp           TURN relay media
```

Restrict SIP signaling to the customer's carrier source networks whenever
possible.

Never expose PostgreSQL, Redis, NATS, FreeSWITCH ESL, or the Media Runtime
control interface directly to the public Internet.

## 7. Validate and start Leamout

Validate the Compose file:

```sh
docker compose --env-file .env -f deploy/compose.yaml config --quiet
```

Start the stack:

```sh
docker compose --env-file .env -f deploy/compose.yaml up -d --build
```

Inspect service state:

```sh
docker compose --env-file .env -f deploy/compose.yaml ps
```

The core deployment includes PostgreSQL, Redis, NATS JetStream, MinIO, the
Leamout API, worker, Media Runtime, OpenSIPS, FreeSWITCH, RTPengine, Coturn, and
Caddy.

Check API health:

```sh
curl -f https://api.voice.example.com/healthz
curl -f https://api.voice.example.com/readyz
```

If the stack is not healthy, inspect logs before configuring external SIP:

```sh
docker compose --env-file .env -f deploy/compose.yaml logs --tail=200
```

## 8. Set API variables

The remaining examples use the public HTTP API.

```sh
export LEAMOUT_API="https://api.voice.example.com/v1"
export LEAMOUT_TOKEN="<organization bearer token>"
```

Every request below uses:

```text
Authorization: Bearer $LEAMOUT_TOKEN
```

The acceptance suites seed deterministic tokens only for isolated tests. A
production deployment needs a real organization and organization-scoped bearer
token. Do not reuse acceptance-test credentials.

## 9. Create the customer SIP trunk

The trunk is the database-backed representation of connectivity that the
customer already owns.

Create a bidirectional trunk:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "customer-carrier",
    "direction": "bidirectional",
    "inbound_enabled": true,
    "codecs": ["PCMU", "PCMA"]
  }'
```

Copy the returned id:

```sh
export TRUNK_ID="<trunk id>"
```

### Add the carrier source network

For IP-authenticated inbound SIP, store the carrier signaling network:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/$TRUNK_ID/source-ips" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cidr":"203.0.113.25/32"}'
```

Replace the example CIDR with the carrier's real signaling network.

### Add the remote SIP endpoint

Store the SIP endpoint supplied by the customer or carrier:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/$TRUNK_ID/endpoints" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "host": "mytrunk.pstn.twilio.com",
    "port": 5061,
    "transport": "tls",
    "direction": "bidirectional"
  }'
```

The hostname above is only an example. Use the exact endpoint from the
customer's SIP provider, PBX, or SBC.

For another customer it could instead be:

```text
sip.provider.example
pbx.customer.example
10.20.30.40
```

Leamout does not insert a Leamout carrier endpoint between the deployment and
that remote peer.

If the peer requires SIP digest authentication, configure the trunk's outbound
and/or inbound authentication through the trunk authentication API. Do not put
carrier secrets in source code.

Validate the resulting trunk configuration:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/$TRUNK_ID/validate" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

Resolve validation errors before continuing.

## 10. Register a customer-owned phone number

Leamout does not purchase or own the number. This record tells the runtime that
a customer-owned number belongs to the configured trunk.

```sh
curl -sS -X POST "$LEAMOUT_API/numbers/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: quickstart-number-1" \
  -d "{
    \"number\": \"+15551234567\",
    \"country_code\": \"US\",
    \"trunk_id\": \"$TRUNK_ID\",
    \"voice_enabled\": true
  }"
```

Use the real E.164 number and country code.

```sh
export NUMBER_ID="<number id>"
```

## 11. Create a Voice Application

A Voice Application binds telephony configuration to the Voice Agent layer.

```sh
curl -sS -X POST "$LEAMOUT_API/voice-applications/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickstart",
    "caller_id": "+15551234567"
  }'
```

```sh
export VOICE_APPLICATION_ID="<voice application id>"
```

Bind the number:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/voice-applications/$VOICE_APPLICATION_ID/bindings" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"phone_number_id\":\"$NUMBER_ID\"}"
```

## 12. Add an OpenAI integration

Create the organization-scoped integration:

```sh
curl -sS -X POST "$LEAMOUT_API/ai-integrations/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "name": "Production OpenAI",
    "secret": "<OpenAI API key>"
  }'
```

```sh
export AI_INTEGRATION_ID="<integration id>"
```

Verify it:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/ai-integrations/$AI_INTEGRATION_ID/verify" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

Do not continue until `connection_state` is `ready`.

## 13. Create an integrated Voice Agent

Create the draft:

```sh
curl -sS -X POST "$LEAMOUT_API/voice-agents/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Quickstart Assistant",
    "engine": "integrated",
    "instructions": "You are a concise and helpful voice assistant.",
    "voice": "alloy",
    "language": "en",
    "engine_config": {},
    "interruption_policy": "allow",
    "recording_policy": "none"
  }'
```

```sh
export VOICE_AGENT_ID="<voice agent id>"
```

Attach the OpenAI integration:

```sh
curl -sS -X PUT \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/providers/realtime" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"provider\": \"openai\",
    \"credential_id\": \"$AI_INTEGRATION_ID\",
    \"config\": {}
  }"
```

Bind the Voice Application:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/bindings" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"voice_application_id\":\"$VOICE_APPLICATION_ID\"}"
```

## 14. Check readiness and activate

Readiness is a server-side runtime gate:

```sh
curl -sS \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/readiness" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

The report should return `ready: true`.

Activate the immutable revision:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/activate" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

## 15. Configure inbound delivery with the customer's carrier

Inbound delivery is provider-specific.

Configure the customer's carrier, PBX, or SBC to originate calls toward the
OpenSIPS listener of this Leamout deployment using the public IP or
customer-controlled hostname supported by that peer.

For example:

```text
telephony.voice.example.com:5061
```

or, where supported:

```text
203.0.113.10:5060
```

These are addresses of the customer's deployment. They are not Leamout carrier
endpoints and are not shared infrastructure provided by Leamout Limited.

The expected inbound path is:

```text
customer carrier / PBX / SBC
    |
    v
OpenSIPS in customer deployment
    |
    v
FreeSWITCH
    |
    v
Media Runtime
    |
    v
Voice Agent
```

Call the customer-owned number from a real phone and watch the runtime:

```sh
docker compose --env-file .env -f deploy/compose.yaml logs -f \
  opensips freeswitch media worker server
```

A successful call should answer, attach exactly one durable Voice Agent session,
exchange bidirectional audio, and terminate without leaving an orphaned session.

## 16. Inspect calls

List recent calls:

```sh
curl -sS "$LEAMOUT_API/calls/?limit=20" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

Inspect one call:

```sh
export CALL_ID="<call id>"

curl -sS "$LEAMOUT_API/calls/$CALL_ID" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

For the first deployment, confirm that the call reached an answered state, a
Voice Agent session was attached, conversation state was persisted, credentials
were not exposed, and both the call and session reached terminal states after
hangup.

## 17. Prove outbound calling

For outbound calls, Leamout resolves the configured trunk endpoint from the
database and routes directly to that customer-provided SIP peer.

```text
POST /calls
    |
    v
trunk + endpoint from PostgreSQL
    |
    v
OpenSIPS
    |
    v
customer SIP endpoint
```

Originate a call:

```sh
curl -sS -X POST "$LEAMOUT_API/calls/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: quickstart-outbound-1" \
  -d "{
    \"application_id\": \"$VOICE_APPLICATION_ID\",
    \"trunk_id\": \"$TRUNK_ID\",
    \"from_uri\": \"+15551234567\",
    \"to_uri\": \"+15557654321\"
  }"
```

Use numbers authorized by the customer's carrier and applicable law.

## 18. Add the composable engine later

After the integrated path is stable, add and verify organization integrations
for Deepgram, Groq, and Cartesia and bind the `stt`, `llm`, and `tts` roles to a
composable Voice Agent.

```text
FreeSWITCH
    |
    v
Deepgram
    |
    v
Groq
    |
    v
Cartesia
    |
    v
FreeSWITCH
```

## Troubleshooting order

Debug from the outside inward:

1. Confirm the local `voice-agent-v1` release gate passes.
2. Confirm the Leamout Compose stack is healthy.
3. Confirm `/healthz` and `/readyz` return HTTP 200.
4. Confirm the database trunk contains the correct remote SIP endpoint.
5. Confirm the source network/authentication matches the customer's carrier.
6. Confirm the carrier can deliver inbound SIP to the customer's deployment.
7. Confirm OpenSIPS resolves and authorizes the intended trunk/endpoint.
8. Confirm FreeSWITCH creates a call channel.
9. Confirm the Media Runtime attaches the audio fork.
10. Confirm the AI integration reports `ready` and the provider session connects.
11. Confirm audio returns through FreeSWITCH and the customer carrier.

Do not debug AI behavior while SIP signaling is still failing.

## Current operational gaps

Some production bootstrap work is still explicit rather than automated:

- certificate provisioning and renewal where TLS is used;
- first organization and bearer-token bootstrap for a fresh install;
- firewall automation;
- a single production smoke command covering every dependency;
- a live-carrier/live-provider acceptance mode.

These are deployment-productization tasks, not reasons to add a managed carrier
or a Leamout-owned SIP network.

## Next reading

- [Architecture](architecture.md)
- [Realtime media plane](media-plane.md)
- [Voice Agent execution plan](voice-agent-execution-plan.md)
- [Storage](storage.md)
- [Provider SDK](provider-sdk.md)

### Upgrading the runtime review fixes

Use Docker Compose 2.33.1 or newer: the signaling network must own the default
route so published SIP ports reach the external listener. Compose passes the
existing `PUBLIC_IP` to that listener for Via and Record-Route, while internal
listeners retain their private identities. UDP/TCP, TLS and WSS retain ports
5060, 5061 and 5062 respectively. Permit bidirectional carrier traffic on those
published ports and verify ACK, BYE and re-INVITE from outside Docker.

The control network now uses `172.29.0.0/24`, a private subnet. Recreate the
Compose network during an upgrade; schedule this while calls are drained.

Webhook delivery and webhook tests accept HTTPS destinations on public
addresses only. Private, loopback, link-local and CGNAT addresses are blocked
at connection time, and environment proxies are disabled for these requests.
Private services should be exposed through an authenticated public endpoint
before configuring a webhook.

Reactivate existing Voice Agents before starting new calls with tools. New
sessions snapshot the executable tool type, endpoint and timeout as well as
the exposed definition. Older snapshots lack those fields and fail closed;
live disable/delete still revokes a tool and signing-secret rotation remains
immediate. Agent edits do not change an already running session's tool target.

Workers periodically recover control of active Voice Agent calls. Existing
provider and audio sessions remain in place, unanswered tool calls are replayed,
and conversation sequence/counters resume from durable turns. A vanished media
session causes the phone call to terminate instead of remaining silently active.
