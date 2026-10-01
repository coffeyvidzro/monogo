package orchestration

import (
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func TestMediaConfigFromSessionUsesDurableSnapshot(t *testing.T) {
	voice := "voice-a"
	language := "en"
	record := sqlc.VoiceAgentSession{
		EngineConfigSnapshot: []byte(`{"model":"realtime-test"}`),
		Engine:               "integrated",
		InstructionsSnapshot: "snapshot instructions",
		Voice:                &voice,
		Language:             &language,
	}
	got := MediaConfigFromSession(record, session.Config{
		Engine:       session.EngineEcho,
		Instructions: "mutable current value",
		Voice:        "voice-b",
		Language:     "fr",
	})

	if got.Engine != session.EngineIntegrated {
		t.Fatalf("engine = %q", got.Engine)
	}
	if string(got.EngineConfig) != string(record.EngineConfigSnapshot) {
		t.Fatalf("engine config = %s", got.EngineConfig)
	}
	if got.Instructions != record.InstructionsSnapshot {
		t.Fatalf("instructions = %q", got.Instructions)
	}
	if got.Voice != voice {
		t.Fatalf("voice = %q", got.Voice)
	}
	if got.Language != language {
		t.Fatalf("language = %q", got.Language)
	}
}
