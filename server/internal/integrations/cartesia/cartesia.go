// Package cartesia defines the streaming text-to-speech integration boundary.
package cartesia

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Config struct {
	APIKey  string
	Model   string
	VoiceID string
}

type Event struct {
	Audio session.AudioFrame
	Err   error
}

type Stream interface {
	Events() <-chan Event
	Close() error
}

type Synthesizer interface {
	Synthesize(context.Context, Config, string, session.AudioFormat) (Stream, error)
}
