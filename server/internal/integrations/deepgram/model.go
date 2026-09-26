// Package deepgram implements Deepgram live speech-to-text streaming.
package deepgram

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

const (
	DefaultEndpoint = "wss://api.deepgram.com/v1/listen"
	DefaultModel    = "nova-3"
)

type Config struct {
	APIKey         string
	Endpoint       string
	Model          string
	Language       string
	InterimResults bool
	Endpointing    time.Duration
}

type Alternative struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
}

type Channel struct {
	Alternatives []Alternative `json:"alternatives"`
}

type Message struct {
	Type        string   `json:"type"`
	Channel     Channel  `json:"channel"`
	IsFinal     bool     `json:"is_final"`
	SpeechFinal bool     `json:"speech_final"`
	Start       float64  `json:"start"`
	Duration    float64  `json:"duration"`
	RequestID   string   `json:"request_id"`
	Metadata    Metadata `json:"metadata"`
}

type Metadata struct {
	RequestID string `json:"request_id"`
}

type Transcript struct {
	Text        string
	IsFinal     bool
	SpeechFinal bool
	Confidence  float64
	Start       time.Duration
	Duration    time.Duration
}

type Event struct {
	Transcript Transcript
	RequestID  string
	Err        error
}

type Stream interface {
	SendAudio(context.Context, session.AudioFrame) error
	Finalize(context.Context) error
	Events() <-chan Event
	Close(context.Context) error
}

type Transcriber interface {
	Start(context.Context, Config, session.AudioFormat) (Stream, error)
}
