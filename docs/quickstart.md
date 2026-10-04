# Leamout quickstart

This guide covers the shortest path from a clean checkout to a working Leamout
Voice Agent.

Leamout is bring-your-own-carrier and bring-your-own-AI. A real deployment
therefore needs two external things that the repository does not provide:

- a SIP carrier, PBX, or SBC that can send and receive calls through Leamout;
- credentials for a supported AI provider.

For the first real call, use the integrated OpenAI Realtime path. It removes the
extra STT and TTS provider boundaries from initial deployment validation. Once
that path is healthy, you can move to the composable Deepgram + Groq + Cartesia
pipeline.

## What success looks like

A working first deployment should complete this path:

```text
real phone
    |
    v
customer-owned SIP carrier
    |
    v
OpenSIPS
    |
    v
FreeSWITCH
    |
    v
Media Runtime
    |
    v
OpenAI Realtime
    |
    v
Voice Agent
    |
    v
assistant audio back to caller
```

The call should also create durable Leamout state for the call, Voice Agent
session, conversation turns, tool executions when used, and the terminal
session result.

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

It proves the complete runtime path, including bidirectional audio, Voice Agent
session creation, immutable configuration snapshots, tool execution,
conversation persistence, and clean call termination.

The local suite deliberately does not use a real carrier or a live AI provider.
It proves the Leamout runtime before external network and provider variables are
introduced.

To preserve the disposable stack after a failure:

```sh
VOICE_AGENT_V1_KEEP_STACK=1 sh tests/voice-agent-v1/run.sh
```

## 2. Prepare a single-node host

Use a Linux host with a public IPv4 address for the first real deployment.
Avoid adding Kubernetes or multi-node placement until one complete inbound and
outbound call works reliably on a single node.

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

## 3. Configure DNS

For a deployment using `example.com`, point these names to the public IPv4 of
the Leamout host:

```text
api.example.com
sip.example.com
turn.example.com
recordings.example.com
```

Replace `example.com` with the deployment domain in the remaining commands.

Allow DNS changes to propagate before troubleshooting SIP or TLS.

## 4. Prepare the environment

Copy the example environment file:

```sh
cp .env.example .env
```

At minimum, replace all example values in `.env`.

A production-style environment contains:

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

Generate independent random values. For example:

```sh
openssl rand -hex 32
```

Use a separate value for each password or token.

`ENCRYPTION_KEY` must be a raw URL-safe base64 encoding of a 16, 24, or 32 byte
AES key. A 32-byte key can be generated with:

```sh
openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'
```

Do not commit `.env`.

AI provider credentials do not belong in `.env`. They are organization-scoped
secrets stored through the AI integration API and encrypted at rest.

## 5. Provision SIP and TURN certificates

The current Compose deployment expects host certificates at:

```text
/etc/leamout/certs/fullchain.pem
/etc/leamout/certs/privkey.pem
```

These files are mounted into OpenSIPS and Coturn.

Provision a valid certificate covering the SIP and TURN hostnames using the
ACME client or certificate-management process for your environment, then place
the resulting certificate and private key at those paths with permissions that
allow only the required services and administrators to read them.

Certificate bootstrap and renewal are not yet automated by this repository.
Treat that as an operational prerequisite for an Internet-facing deployment.

## 6. Open the required network ports

The default deployment publishes the following ports:

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

Restrict signaling and management traffic further when your carrier or network
architecture provides stable source networks.

Never expose PostgreSQL, Redis, NATS, FreeSWITCH ESL, or the Media Runtime
control interface directly to the public Internet.

## 7. Validate and start Leamout

Validate the Compose file first:

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

The core deployment includes:

```text
PostgreSQL
Redis
NATS JetStream
MinIO
Leamout API
Leamout worker
Leamout Media Runtime
OpenSIPS
FreeSWITCH
RTPengine
Coturn
Caddy
```

Check API health through the configured domain:

```sh
curl -f https://api.example.com/healthz
curl -f https://api.example.com/readyz
```

The expected response from `/healthz` is `ok`. `/readyz` should return HTTP
200 only when the API's required dependencies are ready.

If the stack is not healthy, inspect logs before attempting carrier onboarding:

```sh
docker compose --env-file .env -f deploy/compose.yaml logs --tail=200
```

## 8. Set API variables

The remaining examples use the public HTTP API.

```sh
export LEAMOUT_API="https://api.example.com/v1"
export LEAMOUT_TOKEN="<organization bearer token>"
```

Every request below uses:

```text
Authorization: Bearer $LEAMOUT_TOKEN
```

The repository's acceptance suites seed deterministic organization tokens for
isolated tests. A production deployment needs an organization and an
organization-scoped bearer token provisioned through the deployment's identity
or administrative bootstrap flow. Do not reuse acceptance-test credentials.

## 9. Create a SIP trunk

The simplest first carrier is one that supports IP-authenticated SIP. Create a
bidirectional trunk:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-sip",
    "direction": "bidirectional",
    "inbound_enabled": true,
    "codecs": ["PCMU", "PCMA"]
  }'
```

Copy the returned trunk id:

```sh
export TRUNK_ID="<trunk id>"
```

Allow inbound SIP from the carrier signaling address or network:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/$TRUNK_ID/source-ips" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cidr":"203.0.113.25/32"}'
```

Replace the example CIDR with the carrier's real signaling network.

Add the carrier SIP gateway used for outbound calls:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/$TRUNK_ID/endpoints" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "host": "sip.carrier.example",
    "port": 5060,
    "transport": "udp",
    "direction": "bidirectional"
  }'
```

Replace the host, port, transport, and authentication policy with the values
from your carrier.

If the carrier requires SIP digest authentication, configure the trunk's
outbound and/or inbound authentication through the trunk authentication API
before placing calls. Do not store carrier secrets in source code or shell
history.

Validate the trunk configuration:

```sh
curl -sS -X POST "$LEAMOUT_API/trunks/$TRUNK_ID/validate" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

Resolve all validation errors before continuing.

## 10. Register a customer-owned phone number

Leamout does not purchase or own the number. This step tells Leamout that a
number you already control is delivered through the configured SIP trunk.

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

Use your real E.164 number and country code.

Copy the returned id:

```sh
export NUMBER_ID="<number id>"
```

## 11. Create a Voice Application

A Voice Application binds telephony ingress/egress configuration to the Voice
Agent layer.

```sh
curl -sS -X POST "$LEAMOUT_API/voice-applications/" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickstart",
    "caller_id": "+15551234567"
  }'
```

Copy the returned id:

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

Create the organization-scoped integration. The secret is write-only and must
not be returned by later list operations.

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

Copy the returned integration id:

```sh
export AI_INTEGRATION_ID="<integration id>"
```

Verify the credential against the provider:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/ai-integrations/$AI_INTEGRATION_ID/verify" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

Do not continue until `connection_state` is `ready`.

## 13. Create an integrated Voice Agent

Create a draft Voice Agent using the realtime engine:

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

Copy the returned id:

```sh
export VOICE_AGENT_ID="<voice agent id>"
```

Attach the OpenAI integration as the realtime provider:

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

The OpenAI adapter has its own production endpoint and default realtime model;
provider-specific overrides are optional.

Bind the Voice Application to the Voice Agent:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/bindings" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"voice_application_id\":\"$VOICE_APPLICATION_ID\"}"
```

## 14. Check readiness and activate

Readiness is a server-side runtime gate. Do not bypass it.

```sh
curl -sS \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/readiness" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

The report should return `ready: true`. If it does not, use the returned issue
codes, fields, and remediation messages to fix the configuration.

Activate the immutable revision:

```sh
curl -sS -X POST \
  "$LEAMOUT_API/voice-agents/$VOICE_AGENT_ID/activate" \
  -H "Authorization: Bearer $LEAMOUT_TOKEN"
```

Calls should use the activated revision rather than a mutable draft.

## 15. Make the first inbound call

Configure the external carrier so the owned number sends SIP traffic to:

```text
sip.example.com:5060
```

Use the transport and source authentication that match the trunk configuration.

Call the number from a real phone.

The expected runtime path is:

```text
carrier
  -> OpenSIPS
  -> FreeSWITCH
  -> Media Runtime
  -> OpenAI Realtime
  -> Media Runtime
  -> FreeSWITCH
  -> carrier
```

Watch the stack during the call:

```sh
docker compose --env-file .env -f deploy/compose.yaml logs -f \
  opensips freeswitch media worker server
```

A successful first call should answer, attach exactly one durable Voice Agent
session, exchange bidirectional audio, and terminate without leaving an active
or orphaned session behind.

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

For the first deployment, confirm at least:

- the call reached an answered or active state before hangup;
- a durable Voice Agent session was attached;
- conversation turns were persisted;
- provider credentials were not exposed in API responses or snapshots;
- the call and Voice Agent session reached terminal states after hangup.

## 17. Prove outbound calling

After inbound calling works, originate through the same trunk and Voice
Application:

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

Use numbers authorized by your carrier and applicable law.

The called phone should ring and, after answer, enter the same Voice Agent
runtime path as the inbound call.

## 18. Add the composable engine later

Do not add more provider boundaries until the integrated path is stable.

The composable topology is:

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

For that mode, add and verify organization integrations for Deepgram, Groq, and
Cartesia, then bind the `stt`, `llm`, and `tts` roles to a composable Voice
Agent.

## Troubleshooting order

Debug the system from the outside inward.

1. Confirm DNS resolves to the correct public IP.
2. Confirm the Leamout Compose stack is healthy.
3. Confirm `/healthz` and `/readyz` return HTTP 200.
4. Confirm the SIP carrier can reach OpenSIPS.
5. Confirm the trunk source network and endpoint configuration match the carrier.
6. Confirm FreeSWITCH creates a call channel.
7. Confirm the Media Runtime attaches an audio fork.
8. Confirm the AI integration reports `ready`.
9. Confirm the provider session connects.
10. Confirm audio returns through FreeSWITCH and the carrier.

Do not debug AI behavior while SIP signaling is still failing, and do not debug
carrier routing while the local `voice-agent-v1` release gate is failing.

## Current operational gaps

This quickstart documents the runtime that exists today. Some production
bootstrap work is still intentionally explicit rather than automated:

- TLS certificate provisioning and renewal for SIP/TURN;
- first organization and bearer-token bootstrap for a fresh production install;
- firewall automation;
- a single production smoke command covering every dependency;
- a live-carrier/live-provider acceptance mode.

Those are deployment-productization tasks, not reasons to introduce another
runtime architecture.

## Next reading

- [Architecture](architecture.md)
- [Realtime media plane](media-plane.md)
- [Voice Agent execution plan](voice-agent-execution-plan.md)
- [Storage](storage.md)
- [Provider SDK](provider-sdk.md)
