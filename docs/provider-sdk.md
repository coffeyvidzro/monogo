# Provider SDK and Registry

Leamout's provider SDK defines the runtime-facing contracts for AI providers
without exposing provider-specific wire protocols to the Media Runtime.

The SDK lives in `server/internal/providers`.

## Provider kinds

Leamout currently defines four provider kinds:

- `stt` — streaming speech-to-text;
- `llm` — streaming language-model generation;
- `tts` — streaming text-to-speech;
- `realtime` — integrated speech-to-speech sessions.

Each provider exposes a descriptor containing its stable provider id, kind, and
capabilities.

Current built-ins:

| Kind | Provider | Capabilities |
| --- | --- | --- |
| STT | Deepgram | streaming, turn detection |
| LLM | Groq | streaming, tool calling, usage |
| TTS | Cartesia | streaming |
| Realtime | OpenAI | streaming, turn detection, tool calling, usage, barge-in |

## Contract boundary

Provider-specific clients remain responsible for their native wire protocol,
authentication headers, endpoint options, and event framing.

Thin provider adapters translate between those native types and the Leamout
SDK:

```text
Media Runtime / engine
        ↓
Provider SDK contract
        ↓
provider adapter
        ↓
native provider client
        ↓
provider API
```

This keeps runtime orchestration provider-neutral while preserving focused
provider integrations.

## STT

The STT contract accepts PCM frames and emits normalized speech/transcript
events.

```text
SendAudio
   ↓
STT provider
   ↓
speech.started
speech.stopped
transcript.delta
transcript.final
error
```

## LLM

The LLM contract accepts provider-neutral messages and Voice Agent tool
definitions, then emits normalized text/tool/usage events.

Provider-native tool-call framing must not escape the adapter boundary.

## TTS

The TTS contract accepts ordered text fragments and emits PCM frames.

The `more` flag indicates whether another text fragment follows for the same
synthesis response.

## Realtime

The realtime contract owns a complete integrated provider session and returns
the existing provider-neutral `session.Stream`.

Realtime providers therefore support the same Media Runtime event and command
contract as composable engines, including interruption and tool-result
submission.

## Runtime credentials

Provider credentials are supplied through the existing ephemeral
`session.ProviderRuntime` path.

The SDK does not persist secrets and does not move provider credentials into
durable engine configuration.

## Registry

The provider registry stores validated descriptors by provider kind and id.

Media Runtime constructs the built-in registry during startup. Duplicate
provider ids within the same kind or invalid descriptors fail startup.

The registry is intentionally capability metadata, not a billing marketplace
or dynamic Go plugin loader.

## Conformance

The built-in conformance suite verifies:

- each adapter satisfies its SDK interface at compile time;
- each descriptor validates;
- stable provider ids and kinds;
- required capabilities;
- duplicate registration rejection;
- registry lookup/list behavior.

New provider integrations should not be accepted without extending the
conformance suite.
