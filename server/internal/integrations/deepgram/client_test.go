package deepgram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func TestClientStreamsAudioAndTurnTranscript(t *testing.T) {
	received := make(chan []byte, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/v2/listen" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("model") != DefaultModel || r.URL.Query().Get("encoding") != "linear16" {
			t.Errorf("query = %v", r.URL.Query())
		}
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = ws.CloseNow()
		}()
		kind, audio, err := ws.Read(r.Context())
		if err != nil || kind != websocket.MessageBinary {
			return
		}
		received <- audio
		_ = ws.Write(r.Context(), websocket.MessageText, []byte(`{"type":"TurnInfo","request_id":"req-1","event":"EndOfTurn","turn_index":0,"audio_window_start":0.0,"audio_window_end":0.8,"transcript":"hello","end_of_turn_confidence":0.9,"trigger":"model"}`))
		kind, payload, err := ws.Read(r.Context())
		if err != nil || kind != websocket.MessageText || string(payload) != `{"type":"CloseStream"}` {
			return
		}
		_ = ws.Close(websocket.StatusNormalClosure, "complete")
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	stream, err := NewClient(server.Client()).Start(ctx, Config{
		APIKey: "secret",
		Endpoint: "wss" + strings.TrimPrefix(server.URL, "https") + "/v2/listen",
	}, format)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if closeErr := stream.Close(context.Background()); closeErr != nil {
			t.Errorf("close stream: %v", closeErr)
		}
	}()

	want := []byte{1, 0, 2, 0}
	if err := stream.SendAudio(ctx, session.AudioFrame{Data: want, Format: format}); err != nil {
		t.Fatalf("SendAudio() error = %v", err)
	}
	if got := <-received; string(got) != string(want) {
		t.Fatalf("audio = %v", got)
	}
	event := <-stream.Events()
	if event.Err != nil || event.RequestID != "req-1" || event.TurnEvent != "EndOfTurn" ||
		event.Transcript.Text != "hello" || !event.Transcript.SpeechFinal {
		t.Fatalf("event = %+v", event)
	}
}
