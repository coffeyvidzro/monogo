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
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"conversation.item.input_audio_transcription.completed","event_id":"evt-2","transcript":"hello"}`))
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"response.output_audio.delta","delta":"`+encoded+`"}`))
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	format := session.AudioFormat{SampleRateHz: 24000, Channels: 1}
	stream, err := NewClient(server.Client()).Start(ctx, Config{APIKey: "secret", Endpoint: "wss" + strings.TrimPrefix(server.URL, "https"), Voice: "marin"}, session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(), Engine: session.EngineIntegrated, InputFormat: format, OutputFormat: format,
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
	if event := <-stream.Events(); event.Type != session.EventTranscriptFinal ||
		event.Transcript == nil || event.Transcript.Text != "hello" {
		t.Fatalf("transcript event = %+v", event)
	}
	if frame := <-stream.Audio(); string(frame.Data) != string([]byte{3, 0, 4, 0}) {
		t.Fatalf("audio = %v", frame.Data)
	}
}

func TestRealtimeSubmitsToolResultAndContinuesResponse(t *testing.T) {
	received := make(chan ClientEvent, 2)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.CloseNow() }()
		_, _, err = ws.Read(r.Context()) // session.update
		if err != nil {
			return
		}
		for i := 0; i < 2; i++ {
			_, payload, readErr := ws.Read(r.Context())
			if readErr != nil {
				return
			}
			var event ClientEvent
			if json.Unmarshal(payload, &event) != nil {
				return
			}
			received <- event
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	format := session.AudioFormat{SampleRateHz: 24000, Channels: 1}
	stream, err := NewClient(server.Client()).Start(
		ctx,
		Config{APIKey: "secret", Endpoint: "wss" + strings.TrimPrefix(server.URL, "https")},
		session.Config{
			ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
			Engine: session.EngineIntegrated, InputFormat: format, OutputFormat: format,
		},
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = stream.Close(context.Background()) }()

	if err := stream.SubmitToolResult(ctx, session.ToolResult{
		ToolCallID: "call-1",
		Name:       "lookup",
		Content:    `{"ok":true}`,
	}); err != nil {
		t.Fatalf("SubmitToolResult() error = %v", err)
	}

	first := <-received
	if first.Type != "conversation.item.create" ||
		first.Item == nil ||
		first.Item.Type != "function_call_output" ||
		first.Item.CallID != "call-1" ||
		first.Item.Output != `{"ok":true}` {
		t.Fatalf("tool output event = %+v", first)
	}
	second := <-received
	if second.Type != "response.create" {
		t.Fatalf("continuation event = %+v", second)
	}
}
