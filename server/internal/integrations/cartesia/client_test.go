package cartesia

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
)

func TestClientStreamsPCM(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "secret" {
			t.Errorf("API key = %q", r.Header.Get("X-API-Key"))
		}
		if r.URL.Query().Get("cartesia_version") != DefaultAPIVersion {
			t.Errorf("version = %q", r.URL.Query().Get("cartesia_version"))
		}
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer ws.CloseNow()
		_, payload, err := ws.Read(r.Context())
		if err != nil {
			return
		}
		var request GenerationRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			t.Errorf("request: %v", err)
			return
		}
		if request.ModelID != DefaultModel || request.Voice != "voice" {
			t.Errorf("request = %+v", request)
		}
		audio := base64.StdEncoding.EncodeToString([]byte{1, 0, 2, 0})
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"chunk","data":"`+audio+`","status_code":206,"request_id":"req-1"}`))
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"done","done":true,"status_code":206,"request_id":"req-1"}`))
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	stream, err := NewClient(server.Client()).Synthesize(ctx, Config{APIKey: "secret", VoiceID: "voice", Endpoint: "wss" + strings.TrimPrefix(server.URL, "https")}, "hello", format)
	if err != nil {
		t.Fatalf("Synthesize() error = %v", err)
	}
	var events []Event
	for event := range stream.Events() {
		events = append(events, event)
	}
	if len(events) != 2 || string(events[0].Audio.Data) != string([]byte{1, 0, 2, 0}) || !events[1].Done {
		t.Fatalf("events = %+v", events)
	}
}
