package agent

import (
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func TestConversationStatePersistsFinalTurnsAndSummary(t *testing.T) {
	state := newConversationState()
	base := time.Now().UTC()

	state.observe(session.Event{Type: session.EventSpeechStarted, OccurredAt: base})
	state.observe(session.Event{Type: session.EventSpeechStopped, OccurredAt: base.Add(500 * time.Millisecond)})
	user, ok := state.userTurn(session.Event{
		Type:       session.EventTranscriptFinal,
		Transcript: &session.TranscriptEvent{Text: "hello"},
		ProviderID: "user-1",
		OccurredAt: base.Add(550 * time.Millisecond),
	})
	if !ok || user.Sequence != 1 || user.Role != "user" || user.Content != "hello" {
		t.Fatalf("user turn = %+v, ok = %v", user, ok)
	}
	if user.STTLatencyMS == nil || *user.STTLatencyMS != 50 {
		t.Fatalf("STT latency = %v", user.STTLatencyMS)
	}

	state.observe(session.Event{Type: session.EventResponseStarted, OccurredAt: base.Add(700 * time.Millisecond)})
	state.observe(session.Event{
		Type:       session.EventResponseDelta,
		Response:   &session.ResponseEvent{Text: "Hi "},
		OccurredAt: base.Add(750 * time.Millisecond),
	})
	state.observe(session.Event{
		Type:       session.EventResponseDelta,
		Response:   &session.ResponseEvent{Text: "there"},
		OccurredAt: base.Add(800 * time.Millisecond),
	})
	stopped := session.Event{Type: session.EventResponseStopped, ProviderID: "assistant-1", OccurredAt: base.Add(time.Second)}
	state.observe(stopped)
	assistant, ok := state.assistantTurn(stopped)
	if !ok || assistant.Sequence != 2 || assistant.Role != "assistant" || assistant.Content != "Hi there" {
		t.Fatalf("assistant turn = %+v, ok = %v", assistant, ok)
	}

	summary := state.summary()
	if summary.TurnCount != 1 {
		t.Fatalf("turn count = %d", summary.TurnCount)
	}
	if summary.FirstResponseLatencyMS == nil || *summary.FirstResponseLatencyMS != 200 {
		t.Fatalf("first response latency = %v", summary.FirstResponseLatencyMS)
	}
	if summary.AverageTurnLatencyMS == nil || *summary.AverageTurnLatencyMS != 500 {
		t.Fatalf("average turn latency = %v", summary.AverageTurnLatencyMS)
	}
}

func TestConversationStateCountsBargeInOnlyDuringResponse(t *testing.T) {
	state := newConversationState()
	now := time.Now().UTC()

	state.observe(session.Event{Type: session.EventSpeechStarted, OccurredAt: now})
	state.observe(session.Event{Type: session.EventSpeechStopped, OccurredAt: now.Add(100 * time.Millisecond)})
	state.observe(session.Event{Type: session.EventResponseStarted, OccurredAt: now.Add(200 * time.Millisecond)})
	state.observe(session.Event{Type: session.EventSpeechStarted, OccurredAt: now.Add(250 * time.Millisecond)})
	state.observe(session.Event{Type: session.EventSpeechStopped, OccurredAt: now.Add(350 * time.Millisecond)})
	state.observe(session.Event{Type: session.EventSpeechStarted, OccurredAt: now.Add(400 * time.Millisecond)})

	summary := state.summary()
	if summary.InterruptionCount != 2 {
		t.Fatalf("interruption count = %d, want 2", summary.InterruptionCount)
	}
}

func TestConversationStateToolTurnUsesSharedSequence(t *testing.T) {
	state := newConversationState()
	turn := state.toolTurn("lookup", "call-1", `{"ok":true}`, false)
	if turn.Sequence != 1 || turn.Role != "tool" {
		t.Fatalf("tool turn = %+v", turn)
	}
	if turn.ToolName == nil || *turn.ToolName != "lookup" {
		t.Fatalf("tool name = %v", turn.ToolName)
	}
	if turn.ToolCallID == nil || *turn.ToolCallID != "call-1" {
		t.Fatalf("tool call id = %v", turn.ToolCallID)
	}
}
