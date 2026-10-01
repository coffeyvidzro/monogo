package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

type STTStream interface {
	SendAudio(context.Context, session.AudioFrame) error
	Finalize(context.Context) error
	Events() <-chan STTEvent
	Close(context.Context) error
}

type STT interface {
	Descriptor() Descriptor
	StartSTT(
		context.Context,
		Runtime,
		session.AudioFormat,
		string,
	) (STTStream, error)
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

type LLMStream interface {
	Events() <-chan LLMEvent
	Close() error
}

type LLMRequest struct {
	Runtime      Runtime
	Messages     []Message
	Tools        []session.ToolDefinition
	Instructions string
}

type LLM interface {
	Descriptor() Descriptor
	Generate(context.Context, LLMRequest) (LLMStream, error)
}

type TTSEvent struct {
	Audio      session.AudioFrame
	ProviderID string
	Done       bool
	Err        error
}

type TTSStream interface {
	SendText(context.Context, string, bool) error
	Events() <-chan TTSEvent
	Close() error
}

type TTSRequest struct {
	Runtime  Runtime
	Format   session.AudioFormat
	Voice    string
	Language string
}

type TTS interface {
	Descriptor() Descriptor
	StartTTS(context.Context, TTSRequest) (TTSStream, error)
}

type Realtime interface {
	Descriptor() Descriptor
	StartRealtime(
		context.Context,
		Runtime,
		session.Config,
	) (session.Stream, error)
}

type Registry struct {
	descriptors map[string]Descriptor
}

func NewRegistry(descriptors ...Descriptor) (*Registry, error) {
	registry := &Registry{descriptors: make(map[string]Descriptor, len(descriptors))}
	for _, descriptor := range descriptors {
		if err := registry.Register(descriptor); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(descriptor Descriptor) error {
	if r == nil {
		return fmt.Errorf("provider registry is required")
	}
	if err := descriptor.Validate(); err != nil {
		return err
	}
	key := registryKey(descriptor.Kind, descriptor.ID)
	if _, exists := r.descriptors[key]; exists {
		return fmt.Errorf("provider %q already registered for %q", descriptor.ID, descriptor.Kind)
	}
	copyDescriptor := descriptor
	copyDescriptor.Capabilities = append([]Capability(nil), descriptor.Capabilities...)
	r.descriptors[key] = copyDescriptor
	return nil
}

func (r *Registry) Get(kind Kind, id string) (Descriptor, bool) {
	if r == nil {
		return Descriptor{}, false
	}
	descriptor, ok := r.descriptors[registryKey(kind, id)]
	if !ok {
		return Descriptor{}, false
	}
	descriptor.Capabilities = append([]Capability(nil), descriptor.Capabilities...)
	return descriptor, true
}

func (r *Registry) List(kind Kind) []Descriptor {
	if r == nil {
		return nil
	}
	result := make([]Descriptor, 0)
	for _, descriptor := range r.descriptors {
		if descriptor.Kind == kind {
			copyDescriptor := descriptor
			copyDescriptor.Capabilities = append([]Capability(nil), descriptor.Capabilities...)
			result = append(result, copyDescriptor)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func registryKey(kind Kind, id string) string {
	return string(kind) + ":" + strings.TrimSpace(id)
}
