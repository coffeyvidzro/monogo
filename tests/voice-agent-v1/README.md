# Voice Agent v1 realtime release gate

This suite is the mandatory end-to-end release gate for the single-node Leamout Voice Agent Runtime.

It runs PostgreSQL, Redis, NATS, the API, worker, Media Runtime, OpenSIPS,
FreeSWITCH, RTPengine, a synthetic SIP carrier, and a local TLS WebSocket
fixture that implements the minimum OpenAI Realtime protocol needed by the
integrated engine. CI never depends on a live AI provider credential.

## Release contract

A passing run proves one complete autonomous voice call:

```text
synthetic carrier
      ↓ audio
OpenSIPS / FreeSWITCH
      ↓ audio fork
Media Runtime
      ↓
fake realtime provider
      ↓ tool.call
Agent Runtime
      ↓
durable Tool Executor
      ↓ send_dtmf
Agent Runtime
      ↓ tool.result
Media Runtime
      ↓
fake realtime provider
      ↓ assistant transcript + audio
FreeSWITCH
      ↓
synthetic carrier
```

The pull-request gate verifies:

1. BYOC carrier, trunk, number, and Voice Application configuration.
2. Voice Agent creation and binding through the public API.
3. Webhook signing-secret creation, non-disclosure, and rotation.
4. Built-in `send_dtmf` tool creation and provider tool-definition delivery.
5. An outbound call reaches answered state.
6. Exactly one durable active Voice Agent session is created for the call.
7. The FreeSWITCH channel is marked with that durable session id.
8. The realtime provider receives the immutable durable instructions snapshot.
9. Updating the Voice Agent after answer does not mutate the active session snapshot.
10. Carrier audio reaches Media Runtime and the realtime provider.
11. The provider emits a normalized `tool.call`.
12. Leamout executes the built-in tool through the durable tool executor.
13. The provider receives `tool.result` on the same realtime session.
14. Assistant transcript and audio return through Media Runtime and FreeSWITCH.
15. Durable conversation history contains ordered user, tool, and assistant turns.
16. Call hangup completes the durable Voice Agent session.
17. Session summary fields persist turn count, interruption count, first-response latency, and average turn latency.

A failure in the audio, tool, or durable-history path fails the workflow. There
is no optional reduced lifecycle-only mode.

## Run

From the repository root:

```sh
sh tests/voice-agent-v1/run.sh
```

Set `VOICE_AGENT_V1_KEEP_STACK=1` to retain the disposable stack after a
failure for inspection.
