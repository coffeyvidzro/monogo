package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestRealtimeStreamsAudioAndNormalizedEvents(t *testing.T) {
	receivedAudio := make(chan ClientEvent, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("model") != DefaultModel {
			t.Errorf("model = %q", r.URL.Query().Get("model"))
		}
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			if closeErr := ws.CloseNow(); closeErr != nil {
				t.Errorf("close websocket: %v", closeErr)
			}
		}()
		_, updatePayload, err := ws.Read(r.Context())
		if err != nil {
			return
		}
		var update ClientEvent
		_ = json.Unmarshal(updatePayload, &update)
		if update.Type != "session.update" {
			t.Errorf("update = %+v", update)
		}
		_, audioPayload, err := ws.Read(r.Context())
		if err != nil {
			return
		}
		var appendEvent ClientEvent
		_ = json.Unmarshal(audioPayload, &appendEvent)
		receivedAudio <- appendEvent
		encoded := base64.StdEncoding.EncodeToString([]byte{3, 0, 4, 0})
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"input_audio_buffer.speech_started","event_id":"evt-1"}`))
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"response.output_audio.delta","delta":"`+encoded+`"}`))
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	format := session.AudioFormat{SampleRateHz: 24000, Channels: 1}
	stream, err := NewClient(server.Client()).Start(ctx, Config{APIKey: "secret", Endpoint: "wss" + strings.TrimPrefix(server.URL, "https"), Voice: "marin"}, session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(), Engine: session.EngineOpenAIRealtime, InputFormat: format, OutputFormat: format,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if closeErr := stream.Close(context.Background()); closeErr != nil {
			t.Errorf("close stream: %v", closeErr)
		}
	}()
	if err := stream.SendAudio(ctx, session.AudioFrame{Data: []byte{1, 0, 2, 0}, Format: format}); err != nil {
		t.Fatalf("SendAudio() error = %v", err)
	}
	if event := <-receivedAudio; event.Type != "input_audio_buffer.append" || event.Audio == "" {
		t.Fatalf("append event = %+v", event)
	}
	if event := <-stream.Events(); event.Type != session.EventSpeechStarted {
		t.Fatalf("event = %+v", event)
	}
	if frame := <-stream.Audio(); string(frame.Data) != string([]byte{3, 0, 4, 0}) {
		t.Fatalf("audio = %v", frame.Data)
	}
}
