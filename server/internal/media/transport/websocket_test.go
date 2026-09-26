package transport

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/engine/echo"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestWebSocketHandlerAuthenticatesAndEchoesAudio(t *testing.T) {
	cfg := session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine:       session.EngineEcho,
		InputFormat:  session.AudioFormat{SampleRateHz: 16000, Channels: 1},
		OutputFormat: session.AudioFormat{SampleRateHz: 16000, Channels: 1},
	}
	manager, err := session.NewManager(1, time.Minute, map[session.Engine]session.Starter{
		session.EngineEcho: echo.Engine{},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := manager.Start(context.Background(), cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	tokens, err := NewTokenService(strings.Repeat("s", 32))
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	token, err := tokens.Issue(TokenClaims{
		SessionID: cfg.ID, OrganizationID: cfg.OrganizationID, CallID: cfg.CallID, ChannelID: cfg.ChannelID,
	}, time.Minute)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	handler, err := NewWebSocketHandler(tokens, manager, DefaultWebSocketConfig())
	if err != nil {
		t.Fatalf("NewWebSocketHandler() error = %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	address := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=" + token
	connection, _, err := websocket.Dial(ctx, address, nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer connection.CloseNow()
	hello, _ := json.Marshal(map[string]any{
		"type": "hello", "callSid": cfg.ChannelID.String(), "rate": 16000,
		"channels": 1, "encoding": "L16",
	})
	if err := connection.Write(ctx, websocket.MessageText, hello); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	want := []byte{1, 0, 2, 0}
	if err := connection.Write(ctx, websocket.MessageBinary, want); err != nil {
		t.Fatalf("write audio: %v", err)
	}
	messageType, got, err := connection.Read(ctx)
	if err != nil {
		t.Fatalf("read echoed audio: %v", err)
	}
	if messageType != websocket.MessageBinary || string(got) != string(want) {
		t.Fatalf("echo = (%v, %v), want binary %v", messageType, got, want)
	}
}

func TestWebSocketHandlerRejectsInvalidToken(t *testing.T) {
	tokens, _ := NewTokenService(strings.Repeat("s", 32))
	manager, _ := session.NewManager(1, time.Minute, map[session.Engine]session.Starter{
		session.EngineEcho: echo.Engine{},
	})
	handler, _ := NewWebSocketHandler(tokens, manager, DefaultWebSocketConfig())
	server := httptest.NewServer(handler)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, response, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"?token=invalid", nil)
	if err == nil {
		t.Fatal("Dial() error = nil")
	}
	if response == nil || response.StatusCode != 401 {
		t.Fatalf("status = %v, want 401", response)
	}
}
