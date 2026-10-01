package providers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Kind string

const (
	KindSTT      Kind = "stt"
	KindLLM      Kind = "llm"
	KindTTS      Kind = "tts"
	KindRealtime Kind = "realtime"
)

type Capability string

const (
	CapabilityStreaming     Capability = "streaming"
	CapabilityTurnDetection Capability = "turn_detection"
	CapabilityToolCalling   Capability = "tool_calling"
	CapabilityUsage         Capability = "usage"
	CapabilityBargeIn       Capability = "barge_in"
)

type Descriptor struct {
	ID           string
	Kind         Kind
	Capabilities []Capability
}

func (d Descriptor) Validate() error {
	if strings.TrimSpace(d.ID) == "" {
		return fmt.Errorf("provider id is required")
	}

	switch d.Kind {
	case KindSTT, KindLLM, KindTTS, KindRealtime:
	default:
		return fmt.Errorf("unsupported provider kind %q", d.Kind)
	}

	seen := make(map[Capability]struct{}, len(d.Capabilities))
	for _, capability := range d.Capabilities {
		if capability == "" {
			return fmt.Errorf("provider capability is required")
		}
		if _, exists := seen[capability]; exists {
			return fmt.Errorf("duplicate provider capability %q", capability)
		}
		seen[capability] = struct{}{}
	}

	return nil
}

type Runtime struct {
	APIKey string
	Config json.RawMessage
}

type STTEventType string

const (
	STTEventSpeechStarted   STTEventType = "speech.started"
	STTEventSpeechStopped   STTEventType = "speech.stopped"
	STTEventTranscriptDelta STTEventType = "transcript.delta"
	STTEventTranscriptFinal STTEventType = "transcript.final"
	STTEventError           STTEventType = "error"
)

type STTEvent struct {
	Type       STTEventType
	Text       string
	ProviderID string
	Err        error
}

type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type LLMEvent struct {
	ResponseID    string
	TextDelta     string
	ToolCallID    string
	ToolIndex     int
	ToolName      string
	ToolArguments []byte
	InputTokens   int
	OutputTokens  int
	TotalTokens   int
	Done          bool
	Err           error
}

type LLMRequest struct {
	Runtime      Runtime
	Messages     []Message
	Tools        []session.ToolDefinition
	Instructions string
}

type TTSEvent struct {
	Audio      session.AudioFrame
	ProviderID string
	Done       bool
	Err        error
}

type TTSRequest struct {
	Runtime  Runtime
	Format   session.AudioFormat
	Voice    string
	Language string
}
