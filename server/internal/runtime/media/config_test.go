package media

import (
	"testing"
	"time"
)

func TestConfigValidateOpenAIEndpoint(t *testing.T) {
	cfg := Config{
		ListenAddress:    ":8090",
		PublicWebSocket:  "ws://media:8090/v1/audio-forks",
		TokenSecret:      "0123456789abcdef0123456789abcdef",
		ControlToken:     "abcdef0123456789abcdef0123456789",
		TokenTTL:         30 * time.Second,
		MaxSessions:      1,
		ReadLimit:        1024,
		HandshakeTimeout: time.Second,
		DrainTimeout:     time.Second,
		OpenAIEndpoint:   "wss://realtime.example.test/v1/realtime",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid OpenAI endpoint rejected: %v", err)
	}

	cfg.OpenAIEndpoint = "ws://realtime.example.test/v1/realtime"
	if err := cfg.Validate(); err == nil {
		t.Fatal("insecure OpenAI endpoint error = nil")
	}
}
