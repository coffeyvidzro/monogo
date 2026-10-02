package providers

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Provider interface {
	Descriptor() Descriptor
}

type STTStream interface {
	SendAudio(context.Context, session.AudioFrame) error
	Finalize(context.Context) error
	Events() <-chan STTEvent
	Close(context.Context) error
}

type STT interface {
	Provider
	StartSTT(
		context.Context,
		Runtime,
		session.AudioFormat,
		string,
	) (STTStream, error)
}

type LLMStream interface {
	Events() <-chan LLMEvent
	Close() error
}

type LLM interface {
	Provider
	Generate(context.Context, LLMRequest) (LLMStream, error)
}

type TTSStream interface {
	SendText(context.Context, string, bool) error
	Events() <-chan TTSEvent
	Close() error
}

type TTS interface {
	Provider
	StartTTS(context.Context, TTSRequest) (TTSStream, error)
}

type Realtime interface {
	Provider
	StartRealtime(
		context.Context,
		Runtime,
		session.Config,
	) (session.Stream, error)
}
