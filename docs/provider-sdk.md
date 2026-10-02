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
authentication headers, provider-specific configuration, and event framing.

Provider adapters translate between those native types and the Leamout SDK:

```text
Media Runtime / engine
        ↓
Provider Registry
        ↓
Provider SDK contract
        ↓
provider adapter
        ↓
native provider client
        ↓
provider API
```

The engines do not import or branch on individual providers. They resolve an
implementation by provider kind and id through the registry and operate only on
provider-neutral SDK types.

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

## Runtime credentials and configuration

Deployment-level provider configuration supplies the defaults used when Media
Runtime constructs the built-in provider implementations at startup.

A session can override the selected provider, API key, and provider-specific
configuration through `session.ProviderRuntime` for its role:

```text
Composable
├── stt
├── llm
└── tts

Integrated
└── realtime
```

The registry does not persist credentials. Provider-specific configuration is
interpreted by the selected adapter rather than by the composable or integrated
engine.

## Registry

The provider registry owns validated runtime implementations by provider kind
and stable provider id.

Media Runtime constructs the built-in registry once during startup and injects
the same registry into its engines. The composable engine resolves STT, LLM,
and TTS implementations from it; the integrated engine resolves a Realtime
implementation from it.

```text
                     Provider Registry
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
         STT               LLM               TTS
          │                 │                 │
      Deepgram            Groq            Cartesia

                            │
                        Realtime
                            │
                         OpenAI
```

Duplicate provider ids within the same kind, invalid descriptors, or an
implementation that does not satisfy the contract declared by its descriptor
fail registration.

The registry is a runtime implementation registry, not a billing marketplace
or dynamic Go plugin loader.

## Engine responsibilities

Composable orchestration owns provider-neutral behavior such as:

- turn handling;
- response generation lifecycle;
- tool-call/result coordination;
- barge-in and generation fencing;
- normalized usage and response events;
- PCM flow between STT and TTS.

It must not decode Deepgram, Groq, Cartesia, or other provider configuration.

Integrated orchestration selects a Realtime provider and delegates the complete
provider session through the same registry boundary.

## Conformance

The built-in conformance suite verifies:

- each adapter satisfies its SDK interface at compile time;
- each descriptor validates;
- stable provider ids and kinds;
- required capabilities;
- duplicate registration rejection;
- typed implementation lookup;
- registry descriptor lookup/list behavior.

Engine tests use SDK implementations rather than provider-native client
interfaces so provider selection itself is exercised by the tests.

New provider integrations should not be accepted without extending the
conformance suite.
