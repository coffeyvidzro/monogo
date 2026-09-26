package media

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/engine/echo"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/media/transport"
	"github.com/google/uuid"
)

func TestHandlerCreatesAuthenticatedSessionAndUpdatesReadiness(t *testing.T) {
	cfg := validRuntimeConfig()
	cfg.MaxSessions = 1
	manager, err := session.NewManager(1, cfg.TokenTTL, map[session.Engine]session.Starter{
		session.EngineEcho: echo.Engine{},
	})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	tokens, err := transport.NewTokenService(cfg.TokenSecret)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	handler := newHandler(cfg, manager, tokens, http.NotFoundHandler())

	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("initial readiness status = %d", ready.Code)
	}

	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	sessionConfig := session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: session.EngineEcho, InputFormat: format, OutputFormat: format,
	}
	payload, _ := json.Marshal(sessionConfig)
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/internal/v1/sessions",
		bytes.NewReader(payload),
	))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}

	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/internal/v1/sessions",
		bytes.NewReader(payload),
	)
	request.Header.Set("Authorization", "Bearer "+cfg.ControlToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body.String())
	}
	var body createSessionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.WebSocketURL == "" {
		t.Fatalf("response = %q, error = %v", response.Body.String(), err)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}

	notReady := httptest.NewRecorder()
	handler.ServeHTTP(notReady, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil))
	if notReady.Code != http.StatusServiceUnavailable {
		t.Fatalf("capacity readiness status = %d", notReady.Code)
	}
	drainCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Drain(drainCtx); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
}
