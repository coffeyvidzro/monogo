// Package session defines provider-neutral realtime media contracts.
package session

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Engine identifies the media pipeline selected for a session.
type Engine string

const (
	EngineComposable     Engine = "composable"
	EngineOpenAIRealtime Engine = "openai_realtime"
)

// AudioFormat describes an uncompressed PCM stream.
type AudioFormat struct {
	SampleRateHz int
	Channels     int
}

// Validate rejects formats outside the initial media-plane contract.
func (f AudioFormat) Validate() error {
	if f.SampleRateHz != 8000 && f.SampleRateHz != 16000 && f.SampleRateHz != 24000 && f.SampleRateHz != 48000 {
		return fmt.Errorf("unsupported sample rate %d Hz", f.SampleRateHz)
	}
	if f.Channels != 1 {
		return fmt.Errorf("unsupported channel count %d", f.Channels)
	}
	return nil
}

// AudioFrame contains signed 16-bit little-endian PCM captured at CapturedAt.
// Data belongs to the recipient after a successful SendAudio call and must not
// be mutated by the caller.
type AudioFrame struct {
	Data       []byte
	Format     AudioFormat
	CapturedAt time.Time
}

// Validate checks framing invariants without imposing a provider frame size.
func (f AudioFrame) Validate() error {
	if err := f.Format.Validate(); err != nil {
		return err
	}
	if len(f.Data) == 0 {
		return fmt.Errorf("audio frame is empty")
	}
	if len(f.Data)%2 != 0 {
		return fmt.Errorf("PCM16 audio frame has odd byte length %d", len(f.Data))
	}
	return nil
}

// Config is the immutable configuration resolved before a media session starts.
type Config struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CallID         uuid.UUID
	Engine         Engine
	InputFormat    AudioFormat
	OutputFormat   AudioFormat
	Language       string
	Instructions   string
	Voice          string
}

// Validate checks identity and media invariants shared by all engines.
func (c Config) Validate() error {
	if c.ID == uuid.Nil {
		return fmt.Errorf("session id is required")
	}
	if c.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if c.CallID == uuid.Nil {
		return fmt.Errorf("call id is required")
	}
	if c.Engine != EngineComposable && c.Engine != EngineOpenAIRealtime {
		return fmt.Errorf("unsupported engine %q", c.Engine)
	}
	if err := c.InputFormat.Validate(); err != nil {
		return fmt.Errorf("input format: %w", err)
	}
	if err := c.OutputFormat.Validate(); err != nil {
		return fmt.Errorf("output format: %w", err)
	}
	return nil
}

// EventType identifies normalized output from any realtime engine.
type EventType string

const (
	EventSpeechStarted   EventType = "speech.started"
	EventSpeechStopped   EventType = "speech.stopped"
	EventTranscriptDelta EventType = "transcript.delta"
	EventTranscriptFinal EventType = "transcript.final"
	EventResponseStarted EventType = "response.started"
	EventResponseStopped EventType = "response.stopped"
	EventToolCall        EventType = "tool.call"
	EventUsage           EventType = "usage"
	EventError           EventType = "error"
)

// Event is a provider-neutral session event. ProviderPayload is reserved for
// diagnostic metadata and must never be required for domain behavior.
type Event struct {
	Type            EventType
	Text            string
	ProviderID      string
	ProviderPayload []byte
	OccurredAt      time.Time
}

// Stream is one live provider session. Implementations must make Close
// idempotent and close Events when the stream has terminated.
type Stream interface {
	SendAudio(context.Context, AudioFrame) error
	Interrupt(context.Context) error
	Audio() <-chan AudioFrame
	Events() <-chan Event
	Close(context.Context) error
}

// Starter creates a live stream for a validated session configuration.
type Starter interface {
	Start(context.Context, Config) (Stream, error)
}
