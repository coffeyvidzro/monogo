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
	EngineEcho       Engine = "echo"
	EngineComposable Engine = "composable"
	EngineIntegrated Engine = "integrated"
)

// AudioFormat describes an uncompressed PCM stream.
type AudioFormat struct {
	SampleRateHz int `json:"sample_rate_hz"`
	Channels     int `json:"channels"`
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

// ConnectionMetadata binds one authenticated media socket to its tenant,
// call, session, and negotiated PCM format.
type ConnectionMetadata struct {
	SessionID      uuid.UUID
	CallID         uuid.UUID
	ChannelID      uuid.UUID
	OrganizationID uuid.UUID
	RemoteAddress  string
	Format         AudioFormat
}

// Connection is the bidirectional transport presented to a managed session.
type Connection interface {
	Metadata() ConnectionMetadata
	ReceiveAudio(context.Context) (AudioFrame, error)
	SendAudio(context.Context, AudioFrame) error
	ClearPlayback(context.Context) error
	Close() error
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
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	CallID         uuid.UUID   `json:"call_id"`
	VoiceAgentID   uuid.UUID   `json:"voice_agent_id"`
	ChannelID      uuid.UUID   `json:"channel_id"`
	Engine         Engine      `json:"engine"`
	InputFormat    AudioFormat `json:"input_format"`
	OutputFormat   AudioFormat `json:"output_format"`
	Language       string      `json:"language,omitempty"`
	Instructions   string      `json:"instructions,omitempty"`
	Voice          string      `json:"voice,omitempty"`
	Tools          []Tool      `json:"tools,omitempty"`
}

// Tool describes one function available to a realtime engine.
type Tool struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Parameters  []byte    `json:"parameters"`
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
	if c.ChannelID == uuid.Nil {
		return fmt.Errorf("channel id is required")
	}
	if c.Engine != EngineEcho && c.Engine != EngineComposable && c.Engine != EngineIntegrated {
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
	EventResponseDelta   EventType = "response.delta"
	EventResponseStopped EventType = "response.stopped"
	EventToolCall        EventType = "tool.call"
	EventUsage           EventType = "usage"
	EventInterrupted     EventType = "interrupted"
	EventError           EventType = "error"
)

// Event is a provider-neutral session event. ProviderPayload is reserved for
// diagnostic metadata and must never be required for domain behavior.
type Event struct {
	Type            EventType
	Text            string
	ProviderID      string
	ToolName        string
	ProviderPayload []byte
	OccurredAt      time.Time
}

// PublishedEvent is a sequenced, tenant-bound event emitted by a media session.
type PublishedEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	CallID         uuid.UUID `json:"call_id"`
	VoiceAgentID   uuid.UUID `json:"voice_agent_id"`
	Generation     uint64    `json:"generation"`
	Type           EventType `json:"type"`
	Text           string    `json:"text,omitempty"`
	ProviderID     string    `json:"provider_id,omitempty"`
	ToolName       string    `json:"tool_name,omitempty"`
	Payload        []byte    `json:"payload,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// EventPublisher accepts the low-rate lifecycle events produced by a session.
type EventPublisher interface {
	Publish(context.Context, PublishedEvent) error
}

// ToolResult returns one generation-fenced tool execution outcome to an engine.
type ToolResult struct {
	Generation uint64 `json:"generation"`
	CallID     string `json:"call_id"`
	Name       string `json:"name"`
	Content    []byte `json:"content"`
	IsError    bool   `json:"is_error"`
}

// Stream is one live provider session. Implementations must make Close
// idempotent and close Events when the stream has terminated.
type Stream interface {
	SendAudio(context.Context, AudioFrame) error
	Interrupt(context.Context) error
	Audio() <-chan AudioFrame
	Events() <-chan Event
	SendToolResult(context.Context, ToolResult) error
	Close(context.Context) error
}

// Starter creates a live stream for a validated session configuration.
type Starter interface {
	Start(context.Context, Config) (Stream, error)
}
