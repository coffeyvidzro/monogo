# Realtime media plane

The Media Runtime is the realtime execution boundary for live Voice Agent calls.
It is separate from the HTTP control plane and durable worker processes. Live
audio stays on the direct FreeSWITCH ↔ Media Runtime ↔ AI provider path; PCM
frames are never published to NATS or stored in PostgreSQL.

## Package boundaries

- `cmd/media` owns operating-system signal handling for the media process.
- `internal/runtime/media` assembles media dependencies and owns process
  lifecycle, readiness, draining, session creation, and transport credentials.
- `internal/media/session` owns provider-neutral session, audio, engine-profile,
  normalized-event, and live-stream contracts.
- `internal/media/transport` owns the authenticated bidirectional FreeSWITCH
  audio-fork transport.
- `internal/media/engine/composable` owns the STT → LLM → TTS realtime pipeline.
- `internal/media/engine/integrated` adapts end-to-end realtime providers.
- provider packages under `internal/integrations` translate provider protocols
  into the contracts in `internal/media/session`.

Provider payloads may be retained for diagnostics, but runtime behavior must use
normalized session types rather than branching on provider-specific JSON.

## Session configuration

The Agent Runtime resolves a durable Voice Agent session before creating live
media. The media configuration is immutable for the lifetime of the call and
contains:

- organization, call, channel, and Voice Agent session identity;
- selected engine topology;
- engine-owned input/output audio profile;
- instructions, voice, and language snapshots;
- the durable `engine_config_snapshot`.

`engine_config_snapshot` is propagated as validated raw JSON. It is copied from
the durable Voice Agent session rather than reread from the mutable Voice Agent,
so editing an agent while a call is active cannot change that live call.

Provider credentials are runtime secrets and do not belong in
`engine_config_snapshot`.

## Engine audio profiles

Audio requirements are centralized in `internal/media/session`.

Current profiles are:

- `echo`: mono PCM16 at 16 kHz;
- `composable`: mono PCM16 at 16 kHz;
- `integrated`: mono PCM16 at 24 kHz.

The Agent Runtime selects an engine, not a provider sample rate. FreeSWITCH uses
the profile returned by the media contract when starting the audio fork.

## Normalized events

Realtime engines emit typed provider-neutral events:

- `speech.started`;
- `speech.stopped`;
- `transcript.delta` and `transcript.final` with a transcript payload;
- `response.started`, `response.delta`, and `response.stopped` with a
  response payload where applicable;
- `tool.call` with tool-call id, name, and complete JSON arguments;
- `usage` with normalized token counts;
- `error` with a typed failure payload.

Failure payloads distinguish terminal failures from recoverable response-level
failures. Terminal provider/session failures end the media pump. Recoverable
failures remain observable without automatically destroying the call.

Provider framing such as partial function-call argument deltas is not exposed as
a complete normalized tool call.

## Engine topology

The Media Runtime exposes two production topologies:

### Composable

```text
FreeSWITCH
    ↓ PCM
Deepgram Flux
    ↓ transcript / turn events
Groq
    ↓ streamed text
Cartesia
    ↓ PCM
FreeSWITCH
```

Deepgram Flux provides turn detection for the default composable path. There is
no separate local VAD stage.

### Integrated

```text
FreeSWITCH
    ↓ PCM
OpenAI Realtime
    ↓ PCM + normalized events
FreeSWITCH
```

Integrated providers are alternatives to the composable pipeline, not
additional stages inside it.

## Barge-in

The session manager owns provider-neutral interruption behavior.

When user speech starts while response playback is active:

1. stale provider playback is suppressed;
2. the current provider response is interrupted;
3. buffered FreeSWITCH playback is cleared.

This keeps interruption behavior outside individual provider adapters.

## Session creation and attachment

The Media Runtime exposes:

- `GET /livez`;
- `GET /readyz`;
- authenticated `POST /internal/v1/sessions`;
- authenticated/idempotent `DELETE /internal/v1/sessions/{id}`;
- authenticated `GET /internal/v1/sessions/{id}/control` for the Agent Runtime control WebSocket;
- `GET /v1/audio-forks` for the authenticated FreeSWITCH WebSocket attachment.

Two independent timeouts apply:

- `MEDIA_TOKEN_TTL` controls how long a signed, single-use audio-fork token is
  valid;
- `MEDIA_ATTACH_TIMEOUT` controls how long an already-created media session may
  wait for FreeSWITCH to attach.

Token lifetime and session attachment lifetime intentionally do not share one
configuration value.

## Agent control channel

Each live media session may have exactly one Agent Runtime control attachment.

The control WebSocket is distinct from the FreeSWITCH audio WebSocket:

```text
FreeSWITCH ── PCM ───────────────► Media Runtime
Agent Runtime ◄── events/commands ─► Media Runtime
```

Media Runtime sends normalized session events over this channel. Agent Runtime
may currently send:

- `response.interrupt` to cancel the active model response and clear buffered playback;
- `session.stop` to stop the live media session.

Tool-result submission is intentionally not part of this PR. The next layer will
reuse this same channel rather than introducing another realtime transport.

The control path remains in-memory and node-local. It does not publish live
session control traffic through NATS or PostgreSQL.

## Runtime lifecycle

The session manager owns active sessions in process memory. It enforces:

- one audio attachment per session;
- one Agent control attachment per session;
- identity matching across organization, call, channel, and session;
- concurrent-session capacity;
- attachment timeout cleanup;
- graceful drain;
- idempotent session creation by immutable-config equality;
- terminal provider failure propagation.

Active audio/session state is deliberately process-local. PostgreSQL remains the
source of truth for durable Voice Agent sessions, turns, and tool execution, but
is not in the audio hot path.

## Current boundary

The current layer does not yet implement:

- execution of Voice Agent tools;
- tool-result submission back into a live provider stream;
- distributed media-node placement or ownership.

Those features build on this session contract rather than changing the audio
transport or provider boundaries.
