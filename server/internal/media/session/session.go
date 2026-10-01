// Package session defines provider-neutral realtime media contracts.
package session

import (
	"bytes"
	"context"
	"encoding/json"
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

// EngineProfile defines the media format owned by one engine topology.
type EngineProfile struct {
	InputFormat  AudioFormat
	OutputFormat AudioFormat
}

// ProfileForEngine centralizes engine media requirements so callers do not need
// provider-specific sample-rate knowledge.
func ProfileForEngine(engine Engine) (EngineProfile, error) {
	switch engine {
	case EngineEcho, EngineComposable:
		format := AudioFormat{SampleRateHz: 16000, Channels: 1}
		return EngineProfile{InputFormat: format, OutputFormat: format}, nil
	case EngineIntegrated:
		format := AudioFormat{SampleRateHz: 24000, Channels: 1}
		return EngineProfile{InputFormat: format, OutputFormat: format}, nil
	default:
		return EngineProfile{}, fmt.Errorf("unsupported media engine %q", engine)
	}
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
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	CallID         uuid.UUID       `json:"call_id"`
	ChannelID      uuid.UUID       `json:"channel_id"`
	Engine         Engine          `json:"engine"`
	InputFormat    AudioFormat     `json:"input_format"`
	OutputFormat   AudioFormat     `json:"output_format"`
	Language       string          `json:"language,omitempty"`
	Instructions   string          `json:"instructions,omitempty"`
	Voice          string          `json:"voice,omitempty"`
	EngineConfig   json.RawMessage `json:"engine_config,omitempty"`
	Tools          []ToolDefinition `json:"tools,omitempty"`
}

type ToolDefinition struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolResult struct {
	ToolCallID  string `json:"tool_call_id"`
	Name        string `json:"name,omitempty"`
	Content     string `json:"content"`
	IsError     bool   `json:"is_error,omitempty"`
}

// Validate checks identity, media, and immutable configuration invariants.
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
	profile, err := ProfileForEngine(c.Engine)
	if err != nil {
		return err
	}
	if err := c.InputFormat.Validate(); err != nil {
		return fmt.Errorf("input format: %w", err)
	}
	if err := c.OutputFormat.Validate(); err != nil {
		return fmt.Errorf("output format: %w", err)
	}
	if c.InputFormat != profile.InputFormat || c.OutputFormat != profile.OutputFormat {
		return fmt.Errorf(
			"media engine %q requires input %+v and output %+v",
			c.Engine,
			profile.InputFormat,
			profile.OutputFormat,
		)
	}
	if len(c.EngineConfig) != 0 {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(c.EngineConfig, &object); err != nil {
			return fmt.Errorf("engine config must be a JSON object: %w", err)
		}
		if object == nil {
			return fmt.Errorf("engine config must be a JSON object")
		}
	}
	names := make(map[string]struct{}, len(c.Tools))
	for _, tool := range c.Tools {
		if tool.ID == uuid.Nil || tool.Name == "" {
			return fmt.Errorf("media tool id and name are required")
		}
		if _, exists := names[tool.Name]; exists {
			return fmt.Errorf("duplicate media tool name %q", tool.Name)
		}
		names[tool.Name] = struct{}{}
		var parameters map[string]json.RawMessage
		if err := json.Unmarshal(tool.Parameters, &parameters); err != nil || parameters == nil {
			return fmt.Errorf("media tool %q parameters must be a JSON object", tool.Name)
		}
	}
	return nil
}

// Equal compares immutable session configuration, including raw engine config.
func (c Config) Equal(other Config) bool {
	return c.ID == other.ID &&
		c.OrganizationID == other.OrganizationID &&
		c.CallID == other.CallID &&
		c.ChannelID == other.ChannelID &&
		c.Engine == other.Engine &&
		c.InputFormat == other.InputFormat &&
		c.OutputFormat == other.OutputFormat &&
		c.Language == other.Language &&
		c.Instructions == other.Instructions &&
		c.Voice == other.Voice &&
		bytes.Equal(c.EngineConfig, other.EngineConfig) &&
		toolDefinitionsEqual(c.Tools, other.Tools)
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
	EventError           EventType = "error"
)

type TranscriptEvent struct {
	Text string `json:"text"`
}

type ResponseEvent struct {
	Text string `json:"text,omitempty"`
}

type ToolCallEvent struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type UsageEvent struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type FailureEvent struct {
	Source   string `json:"source,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
	Terminal bool   `json:"terminal"`
}

// Event is a provider-neutral session event. ProviderPayload is reserved for
// diagnostics and must never be required for runtime behavior.
type Event struct {
	Type            EventType        `json:"type"`
	Transcript      *TranscriptEvent `json:"transcript,omitempty"`
	Response        *ResponseEvent   `json:"response,omitempty"`
	ToolCall        *ToolCallEvent   `json:"tool_call,omitempty"`
	Usage           *UsageEvent      `json:"usage,omitempty"`
	Failure         *FailureEvent    `json:"failure,omitempty"`
	ProviderID      string           `json:"provider_id,omitempty"`
	ProviderPayload []byte           `json:"-"`
	OccurredAt      time.Time        `json:"occurred_at"`
}


type CommandType string

const (
	CommandInterrupt CommandType = "response.interrupt"
	CommandStop      CommandType = "session.stop"
	CommandToolResult CommandType = "tool.result"
)

// Command is a provider-neutral control instruction sent by Agent Runtime to
// one live Media Runtime session.
type Command struct {
	Type       CommandType `json:"type"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

func (c Command) Validate() error {
	switch c.Type {
	case CommandInterrupt, CommandStop:
		if c.ToolResult != nil {
			return fmt.Errorf("media command %q does not accept a tool result", c.Type)
		}
		return nil
	case CommandToolResult:
		if c.ToolResult == nil {
			return fmt.Errorf("tool_result is required")
		}
		if c.ToolResult.ToolCallID == "" {
			return fmt.Errorf("tool_call_id is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported media command %q", c.Type)
	}
}

// Stream is one live provider session. Implementations must make Close
// idempotent and close Events when the stream has terminated.
type Stream interface {
	SendAudio(context.Context, AudioFrame) error
	Interrupt(context.Context) error
	SubmitToolResult(context.Context, ToolResult) error
	Audio() <-chan AudioFrame
	Events() <-chan Event
	Close(context.Context) error
}

// Starter creates a live stream for a validated session configuration.
type Starter interface {
	Start(context.Context, Config) (Stream, error)
}

func toolDefinitionsEqual(left, right []ToolDefinition) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].ID != right[i].ID ||
			left[i].Name != right[i].Name ||
			left[i].Description != right[i].Description ||
			!bytes.Equal(left[i].Parameters, right[i].Parameters) {
			return false
		}
	}
	return true
}
