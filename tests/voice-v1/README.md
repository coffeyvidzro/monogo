# Voice v1 acceptance suite

This suite is the release gate for roadmap milestone **1. Self-hosted programmable voice**.

It runs against the real Docker Compose telecom stack and uses a synthetic SIP carrier plus a local HTTPS webhook receiver. It does not replace call/media behavior with mocks. A `2xx` response is not enough when an observable media-side assertion is available.

## Acceptance contract

The runner reports one result for each milestone capability:

1. Deploy Leamout.
2. Configure a SIP endpoint/provider.
3. Create a voice application.
4. Receive an inbound call.
5. Originate an outbound call.
6. Answer/hang up.
7. Transfer.
8. Hold/resume.
9. Play audio.
10. Record.
11. Create/manage conferences.
12. Receive normalized call events.
13. Query call state.
14. Receive webhooks.
15. Inspect call/media health.
16. Restart components without corrupting state.

The process exits non-zero unless all sixteen checks pass.

## What is real

- PostgreSQL, Redis, NATS, OpenSIPS, RTPengine, FreeSWITCH, API, and worker use the normal deployment definitions.
- Outbound calls route `FreeSWITCH -> OpenSIPS -> synthetic carrier FreeSWITCH`.
- Inbound calls route `synthetic carrier FreeSWITCH -> OpenSIPS -> FreeSWITCH` using source-IP carrier authentication and a DID/application binding.
- Call controls execute through the public API and are checked against persisted call state and/or the live FreeSWITCH channel.
- Recording assertions wait for worker-consumed `RECORD_START`/`RECORD_STOP` lifecycle events.
- Webhooks use HTTPS with a throwaway CA generated for each run. The runner verifies the HMAC signature returned by Leamout.
- Restart assertions restart API, worker, and FreeSWITCH and verify readiness recovery plus durable call state.
- Conference acceptance requires a matching live FreeSWITCH conference; persistence-only conference state is not considered sufficient.

## Test-only bootstrap

One fact is inserted directly into the isolated test database because it does not currently have a public provisioning API:

- the deterministic organization bearer token used by the runner.

The test DID and its carrier ownership are configured through the public number API. Carrier connections, source IPs, trunks, trunk endpoints, voice applications, bindings, calls, recordings, conferences, and webhooks are likewise exercised through their public APIs.

All bootstrap data lives in the disposable Compose database and is removed by the default cleanup.

## Run

Requirements:

- Docker with Compose v2;
- OpenSSL;
- Python 3.

From the repository root:

```sh
sh tests/acceptance/voice-v1/run.sh
```

Set `VOICE_V1_KEEP_STACK=1` to retain the stack after the run for debugging:

```sh
VOICE_V1_KEEP_STACK=1 sh tests/acceptance/voice-v1/run.sh
```

The default fixture numbers are non-routable test identities used only inside the synthetic stack. Override `VOICE_V1_DID` and `VOICE_V1_CALLER` if needed.

## Interpreting failures

A failing capability is a product gap or a broken deployment path until proven otherwise. The suite intentionally keeps independent checks running so the final matrix shows more than the first failure.

In particular, conference APIs must be backed by observable FreeSWITCH conference state to satisfy item 11; returning success from a persistence-only handler is not accepted as Voice v1 completion.
