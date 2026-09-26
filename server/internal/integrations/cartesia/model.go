// Package cartesia implements Cartesia streaming text-to-speech.
package cartesia

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

const (
	DefaultEndpoint   = "wss://api.cartesia.ai/tts/websocket"
	DefaultAPIVersion = "2026-08-14"
	DefaultModel      = "sonic-3.6"
)

type Config struct {
	APIKey     string
	Endpoint   string
	APIVersion string
	Model      string
	VoiceID    string
	Language   string
}

type OutputFormat struct {
	Container  string `json:"container"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

type GenerationRequest struct {
	ModelID      string       `json:"model_id"`
	Transcript   string       `json:"transcript"`
	Voice        string       `json:"voice"`
	Language     string       `json:"language,omitempty"`
	ContextID    string       `json:"context_id"`
	OutputFormat OutputFormat `json:"output_format"`
	Continue     bool         `json:"continue"`
}

type Response struct {
	Type       string `json:"type"`
	Data       string `json:"data"`
	Done       bool   `json:"done"`
	StatusCode int    `json:"status_code"`
	ContextID  string `json:"context_id"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	ErrorCode  string `json:"error_code"`
	RequestID  string `json:"request_id"`
}

type Event struct {
	Audio     session.AudioFrame
	Done      bool
	RequestID string
	Err       error
}

type Stream interface {
	Events() <-chan Event
	Close() error
}

// TextStream accepts ordered text fragments for one synthesis context. More
// must be true while another fragment will follow and false for the final
// fragment.
type TextStream interface {
	Stream
	SendText(context.Context, string, bool) error
}

type Synthesizer interface {
	Synthesize(context.Context, Config, string, session.AudioFormat) (Stream, error)
}

type StreamingSynthesizer interface {
	StartSynthesis(context.Context, Config, session.AudioFormat) (TextStream, error)
}
