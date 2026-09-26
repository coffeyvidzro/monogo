package voiceai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestMediaClientCreatesAndStopsSession(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef"
	sessionID := uuid.New()
	var created bool
	var stopped bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/internal/v1/sessions":
			var cfg session.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, "invalid", http.StatusBadRequest)
				return
			}
			if cfg.ID != sessionID {
				http.Error(w, "wrong id", http.StatusBadRequest)
				return
			}
			created = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"websocket_url": "ws://media.internal/v1/audio-forks?token=test",
			})
		case r.Method == http.MethodDelete &&
			r.URL.Path == "/internal/v1/sessions/"+sessionID.String():
			stopped = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := DefaultConfig(server.URL, token)
	cfg.RequestTimeout = time.Second
	client, err := newMediaClient(cfg)
	if err != nil {
		t.Fatalf("newMediaClient() error = %v", err)
	}

	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	websocketURL, err := client.CreateSession(context.Background(), session.Config{
		ID:             sessionID,
		OrganizationID: uuid.New(),
		CallID:         uuid.New(),
		ChannelID:      uuid.New(),
		Engine:         session.EngineComposable,
		InputFormat:    format,
		OutputFormat:   format,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if websocketURL == "" || !created {
		t.Fatalf("CreateSession() websocketURL = %q, created = %v", websocketURL, created)
	}

	if err := client.StopSession(context.Background(), sessionID); err != nil {
		t.Fatalf("StopSession() error = %v", err)
	}
	if !stopped {
		t.Fatal("StopSession() did not reach media control endpoint")
	}
}
