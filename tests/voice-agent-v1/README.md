# Voice Agent v1 lifecycle acceptance

This suite is the release gate for the first Voice Agent control-plane and call-lifecycle path.

It runs the normal Leamout PostgreSQL, Redis, NATS, API, worker, media process,
OpenSIPS, FreeSWITCH, and RTPengine. A synthetic SIP carrier terminates the call.
A local TLS WebSocket fixture implements the minimum OpenAI Realtime protocol
needed by the integrated engine, so CI never uses a live AI provider credential.

## Default release contract

Pull-request CI verifies:

1. BYOC carrier, trunk, number, and Voice Application configuration.
2. Voice Agent creation and binding through the public API.
3. Webhook tool signing-secret creation, non-disclosure, and rotation.
4. Built-in tool definition validation.
5. An outbound call reaches answered state.
6. Exactly one durable active Voice Agent session is created for the call.
7. The FreeSWITCH channel is marked with that durable session id.
8. The integrated provider receives the durable instructions snapshot.
9. Updating the Voice Agent after answer does not mutate the active session snapshot.
10. Call hangup completes the durable Voice Agent session.

## Known media-handshake gap

The bidirectional FreeSWITCH -> media -> realtime-provider audio round-trip is
implemented as a strict check but is not a pull-request merge gate yet.

Current diagnostics show that FreeSWITCH reaches the media service, the signed
token is accepted, and the WebSocket upgrade succeeds, but the first media
protocol frame is rejected during the hello handshake. This remains a product
runtime gap and is tracked separately from this lifecycle acceptance gate.

Run the strict audio check explicitly with:

```sh
VOICE_AGENT_V1_REQUIRE_AUDIO=1 sh tests/voice-agent-v1/run.sh
```

The GitHub workflow also exposes a manual `require_audio` input.

The suite does **not** claim live provider tool-call dispatch yet. PR #96 made
tool execution durable and secure behind the orchestration boundary, but the
media session manager does not yet dispatch provider `tool.call` events into
that boundary.

## Run

From the repository root:

```sh
sh tests/voice-agent-v1/run.sh
```

Set `VOICE_AGENT_V1_KEEP_STACK=1` to retain the disposable stack after a
failure for inspection.
