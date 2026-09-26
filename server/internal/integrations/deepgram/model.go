// Package deepgram implements Deepgram Flux conversational speech recognition.
package deepgram

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

const (
	DefaultEndpoint = "wss://api.deepgram.com/v2/listen"
	DefaultModel    = "flux-general-en"
)

type Config struct {
	APIKey             string
	Endpoint           string
	Model              string
	LanguageHints      []string
	EOTThreshold       *float64
	EagerEOTThreshold  *float64
	EOTTimeout         time.Duration
}

type Message struct {
	Type                string  `json:"type"`
	Event               string  `json:"event"`
	RequestID           string  `json:"request_id"`
	TurnIndex           int     `json:"turn_index"`
	AudioWindowStart    float64 `json:"audio_window_start"`
	AudioWindowEnd      float64 `json:"audio_window_end"`
	Transcript          string  `json:"transcript"`
	EndOfTurnConfidence float64 `json:"end_of_turn_confidence"`
	Trigger             string  `json:"trigger"`
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
	TurnEvent  string
	TurnIndex  int
	Trigger    string
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
