package integrated

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestEngineStartsOpenAIRealtimeWithSessionVoice(t *testing.T) {
	updateReceived := make(chan openai.ClientEvent, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.CloseNow() }()

		_, payload, err := ws.Read(r.Context())
		if err != nil {
			return
		}
		var update openai.ClientEvent
		if err := json.Unmarshal(payload, &update); err != nil {
			return
		}
		updateReceived <- update
		<-r.Context().Done()
	}))
	defer server.Close()

	format := session.AudioFormat{SampleRateHz: 24000, Channels: 1}
	cfg := session.Config{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CallID:         uuid.New(),
		ChannelID:      uuid.New(),
		Engine:         session.EngineIntegrated,
		InputFormat:    format,
		OutputFormat:   format,
		Instructions:   "Be concise.",
		Voice:          "marin",
	}
	engine := Engine{
		Client: openai.NewClient(server.Client()),
		Config: openai.Config{
			APIKey:   "secret",
			Endpoint: "wss" + strings.TrimPrefix(server.URL, "https"),
			Voice:    "cedar",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := engine.Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = stream.Close(context.Background()) }()

	select {
	case update := <-updateReceived:
		if update.Session == nil {
			t.Fatal("session update = nil")
		}
		if update.Session.Instructions != cfg.Instructions {
			t.Fatalf("instructions = %q", update.Session.Instructions)
		}
		if update.Session.Audio.Output.Voice != cfg.Voice {
			t.Fatalf("voice = %q", update.Session.Audio.Output.Voice)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for session update")
	}
}

func TestEngineRejectsNonIntegratedSession(t *testing.T) {
	format := session.AudioFormat{SampleRateHz: 24000, Channels: 1}
	_, err := (Engine{}).Start(context.Background(), session.Config{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CallID:         uuid.New(),
		ChannelID:      uuid.New(),
		Engine:         session.EngineEcho,
		InputFormat:    format,
		OutputFormat:   format,
	})
	if err == nil {
		t.Fatal("Start() error = nil")
	}
}
