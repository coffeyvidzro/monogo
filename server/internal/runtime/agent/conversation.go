package agent

import (
	"strings"
	"sync"
	"time"

	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type conversationState struct {
	mu sync.Mutex

	nextSequence int32

	speechStartedAt time.Time
	speechStoppedAt time.Time
	responseStartedAt time.Time
	responseActive bool
	assistantText strings.Builder

	turnCount int32
	interruptionCount int32
	firstResponseLatency *time.Duration
	totalTurnLatency time.Duration
	turnLatencyCount int32
}

func newConversationState() *conversationState {
	return &conversationState{nextSequence: 1}
}

func (s *conversationState) observe(event session.Event) {
	at := event.OccurredAt
	if at.IsZero() {
		at = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch event.Type {
	case session.EventSpeechStarted:
		if s.responseActive {
			s.interruptionCount++
		}
		s.speechStartedAt = at
	case session.EventSpeechStopped:
		s.speechStoppedAt = at
	case session.EventResponseStarted:
		s.responseStartedAt = at
		s.responseActive = true
		s.assistantText.Reset()
		if !s.speechStoppedAt.IsZero() {
			latency := at.Sub(s.speechStoppedAt)
			if latency >= 0 && s.firstResponseLatency == nil {
				copy := latency
				s.firstResponseLatency = &copy
			}
		}
	case session.EventResponseDelta:
		if event.Response != nil {
			s.assistantText.WriteString(event.Response.Text)
		}
	case session.EventResponseStopped:
		s.responseActive = false
	}
}

func (s *conversationState) userTurn(event session.Event) (conversations.CreateTurnRequest, bool) {
	if event.Transcript == nil {
		return conversations.CreateTurnRequest{}, false
	}
	text := strings.TrimSpace(event.Transcript.Text)
	if text == "" {
		return conversations.CreateTurnRequest{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	sequence := s.nextSequence
	s.nextSequence++

	var providerID *string
	if event.ProviderID != "" {
		value := event.ProviderID
		providerID = &value
	}
	var speechStartedAt, speechEndedAt *time.Time
	if !s.speechStartedAt.IsZero() {
		value := s.speechStartedAt
		speechStartedAt = &value
	}
	if !s.speechStoppedAt.IsZero() {
		value := s.speechStoppedAt
		speechEndedAt = &value
	}
	var sttLatency *int32
	if !s.speechStoppedAt.IsZero() && !event.OccurredAt.IsZero() {
		value := durationMilliseconds(event.OccurredAt.Sub(s.speechStoppedAt))
		sttLatency = &value
	}
	return conversations.CreateTurnRequest{
		Sequence: sequence,
		Role: "user",
		Content: text,
		ProviderID: providerID,
		SpeechStartedAt: speechStartedAt,
		SpeechEndedAt: speechEndedAt,
		STTLatencyMS: sttLatency,
	}, true
}

func (s *conversationState) assistantTurn(event session.Event) (conversations.CreateTurnRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	text := strings.TrimSpace(s.assistantText.String())
	s.assistantText.Reset()
	if text == "" {
		return conversations.CreateTurnRequest{}, false
	}
	sequence := s.nextSequence
	s.nextSequence++
	s.turnCount++
	if !s.speechStoppedAt.IsZero() && !event.OccurredAt.IsZero() {
		latency := event.OccurredAt.Sub(s.speechStoppedAt)
		if latency >= 0 {
			s.totalTurnLatency += latency
			s.turnLatencyCount++
		}
	}

	var providerID *string
	if event.ProviderID != "" {
		value := event.ProviderID
		providerID = &value
	}
	var llmTTFT, turnLatency *int32
	if !s.speechStoppedAt.IsZero() && !s.responseStartedAt.IsZero() {
		value := durationMilliseconds(s.responseStartedAt.Sub(s.speechStoppedAt))
		llmTTFT = &value
	}
	if !s.speechStoppedAt.IsZero() && !event.OccurredAt.IsZero() {
		value := durationMilliseconds(event.OccurredAt.Sub(s.speechStoppedAt))
		turnLatency = &value
	}
	return conversations.CreateTurnRequest{
		Sequence: sequence,
		Role: "assistant",
		Content: text,
		ProviderID: providerID,
		LLMTTFTMS: llmTTFT,
		TurnLatencyMS: turnLatency,
	}, true
}

func (s *conversationState) toolTurn(name, callID, content string, isError bool) conversations.CreateTurnRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	sequence := s.nextSequence
	s.nextSequence++
	toolName := name
	toolCallID := callID
	metadata := []byte(`{"is_error":false}`)
	if isError {
		metadata = []byte(`{"is_error":true}`)
	}
	return conversations.CreateTurnRequest{
		Sequence: sequence,
		Role: "tool",
		Content: content,
		ToolName: &toolName,
		ToolCallID: &toolCallID,
		Metadata: metadata,
	}
}

func (s *conversationState) summary() conversationSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := conversationSummary{
		TurnCount: s.turnCount,
		InterruptionCount: s.interruptionCount,
	}
	if s.firstResponseLatency != nil {
		value := durationMilliseconds(*s.firstResponseLatency)
		result.FirstResponseLatencyMS = &value
	}
	if s.turnLatencyCount > 0 {
		value := int32(s.totalTurnLatency.Milliseconds() / int64(s.turnLatencyCount))
		result.AverageTurnLatencyMS = &value
	}
	return result
}

type conversationSummary struct {
	TurnCount int32
	InterruptionCount int32
	FirstResponseLatencyMS *int32
	AverageTurnLatencyMS *int32
}

func durationMilliseconds(value time.Duration) int32 {
	if value < 0 {
		return 0
	}
	ms := value.Milliseconds()
	const maxInt32 = int64(1<<31 - 1)
	if ms > maxInt32 {
		return int32(maxInt32)
	}
	return int32(ms)
}
