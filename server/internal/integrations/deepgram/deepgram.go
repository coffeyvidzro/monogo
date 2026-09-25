// Package deepgram defines the streaming speech-to-text integration boundary.
package deepgram

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Config struct {
	APIKey   string
	Model    string
	Language string
}

type Transcript struct {
	Text        string
	IsFinal     bool
	SpeechFinal bool
	Confidence  float64
}

type Event struct {
	Transcript Transcript
	Err        error
}

type Stream interface {
	SendAudio(context.Context, session.AudioFrame) error
	Events() <-chan Event
	Close(context.Context) error
}

type Transcriber interface {
	Start(context.Context, Config, session.AudioFormat) (Stream, error)
}
