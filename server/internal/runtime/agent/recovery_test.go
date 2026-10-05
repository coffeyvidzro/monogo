package agent

import (
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/google/uuid"
)

func TestWorkerRestartReattachesExistingControlWithoutRestartingAudio(t *testing.T) {
	db := newLifecycleDB()
	media := newLifecycleMediaServer(t)
	fs := newLifecycleFreeSWITCHServer(t, false)
	first := newLifecycleRuntime(t, db, media.URL, fs)
	call := sqlc.Call{ID: db.callID, OrganizationID: db.organizationID, VoiceAgentID: &db.agent.ID}
	event := calling.LifecycleEvent{Type: calling.LifecycleAnswered, ChannelID: uuid.NewString()}
	if err := first.HandleLifecycle(t.Context(), call, event); err != nil {
		t.Fatal(err)
	}
	first.mu.Lock()
	control := first.controls[db.sessionID]
	first.mu.Unlock()
	if err := control.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.freeSwitch.Close(); err != nil {
		t.Fatal(err)
	}
	second := newLifecycleRuntime(t, db, media.URL, fs)
	if err := second.HandleLifecycle(t.Context(), call, event); err != nil {
		t.Fatal(err)
	}
	second.mu.Lock()
	recovered := second.controls[db.sessionID]
	state := second.states[db.sessionID]
	second.mu.Unlock()
	if recovered == nil || state == nil {
		t.Fatal("worker did not recover control and conversation state")
	}
	_ = recovered.Close()
	if media.createCount() != 1 || fs.audioForkStarts() != 1 || db.createCount() != 1 {
		t.Fatal("recovery duplicated media, audio fork or durable session")
	}
}

func TestConversationRecoveryContinuesSequenceAndSummary(t *testing.T) {
	latency := int32(250)
	turns := []sqlc.VoiceAgentTurn{
		{Sequence: 1, Role: "user"},
		{Sequence: 3, Role: "assistant", TurnLatencyMs: &latency, Metadata: []byte(`{"interruption_count":2,"first_response_latency_ms":75}`)},
		{Sequence: 5, Role: "tool"},
	}
	state := restoreConversationState(sqlc.VoiceAgentSession{}, turns)
	turn := state.toolTurn("lookup", "tool-2", "{}", false)
	if turn.Sequence != 6 {
		t.Fatalf("sequence = %d, want 6", turn.Sequence)
	}
	summary := state.summary()
	if summary.TurnCount != 1 || summary.InterruptionCount != 2 || summary.FirstResponseLatencyMS == nil || *summary.FirstResponseLatencyMS != 75 || summary.AverageTurnLatencyMS == nil || *summary.AverageTurnLatencyMS != 250 {
		t.Fatalf("lost durable summary: %+v", summary)
	}
	if state.totalTurnLatency != 250*time.Millisecond {
		t.Fatal("latency was not restored")
	}
}
