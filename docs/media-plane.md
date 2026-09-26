# Realtime media plane

The media plane is kept separate from the HTTP control plane and durable job
worker. It will carry live call audio between FreeSWITCH and one selected AI
engine without publishing audio frames to NATS or storing them in PostgreSQL.

## Package boundaries

- `cmd/media` owns operating-system signal handling for the media process.
- `internal/runtime/media` assembles media dependencies and owns their process
  lifecycle.
- `internal/media/session` contains the provider-neutral session, audio, and
  event contracts used by orchestration code.
- `internal/media/transport` contains the authenticated, bidirectional
  FreeSWITCH transport contract.
- `internal/media/vad` contains the voice-activity detector contract. The first
  implementation is intended to use Silero through ONNX Runtime.
- `internal/integrations/deepgram` contains streaming speech-to-text contracts.
- `internal/integrations/groq` contains streaming language-model contracts.
- `internal/integrations/cartesia` contains streaming text-to-speech contracts.
- `internal/integrations/openai` contains the end-to-end realtime engine
  contract.

Provider packages translate remote protocols into the types in
`internal/media/session`. Domain and runtime packages must not branch on raw
provider payloads. The optional provider payload on a normalized event is for
diagnostics only.

All streaming contracts expose receive-only event or audio channels. Closing a
stream closes those channels, and terminal provider failures are delivered on
the provider event channel before it closes. This keeps asynchronous failures
observable without allowing callers to write into an adapter's queues.

## Engine topology

The composable engine connects Silero turn detection, Deepgram transcription,
Groq generation, and Cartesia synthesis. The OpenAI realtime engine is an
alternative end-to-end path. A media session selects exactly one of these
engines; OpenAI realtime is not an extra stage in the composable pipeline.

The initial PCM contract is mono signed 16-bit little-endian audio at 8, 16,
24, or 48 kHz. Provider adapters are responsible for rejecting unsupported
formats or resampling at their boundary. A caller must not mutate frame data
after successfully handing a frame to a stream.

## Runtime status

The media command exposes liveness, readiness, an authenticated internal
session-creation endpoint, and a WebSocket endpoint for FreeSWITCH audio forks.
Session credentials are HMAC-signed, short-lived, and single use. The runtime
enforces a concurrent-session limit and stops accepting work before draining
active connections during shutdown.

The initial engine is a bounded echo implementation used to prove full-duplex
PCM transport. FreeSWITCH builds the MIT-licensed `lazyboson/mod_audio_fork` at
an immutable commit and loads it with low-latency buffering. Run the
`tests/media-v1` acceptance suite to verify generated FreeSWITCH audio travels
through the Go worker and returns to FreeSWITCH playback.

External AI providers are not connected yet. Silero VAD and the composable and
OpenAI engines will be introduced behind the tested session boundary after the
transport foundation is stable.
