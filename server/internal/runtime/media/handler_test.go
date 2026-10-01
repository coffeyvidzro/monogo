package media

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	manager, err := session.NewManager(1, cfg.AttachTimeout, map[session.Engine]session.Starter{
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

	replayRequest := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/internal/v1/sessions",
		bytes.NewReader(payload),
	)
	replayRequest.Header.Set("Authorization", "Bearer "+cfg.ControlToken)
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusCreated {
		t.Fatalf("replayed create status = %d, body = %s", replay.Code, replay.Body.String())
	}

	notReady := httptest.NewRecorder()
	handler.ServeHTTP(notReady, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil))
	if notReady.Code != http.StatusServiceUnavailable {
		t.Fatalf("capacity readiness status = %d", notReady.Code)
	}

	stopRequest := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"/internal/v1/sessions/"+sessionConfig.ID.String(),
		nil,
	)
	stopRequest.Header.Set("Authorization", "Bearer "+cfg.ControlToken)
	stopped := httptest.NewRecorder()
	handler.ServeHTTP(stopped, stopRequest)
	if stopped.Code != http.StatusNoContent {
		t.Fatalf("stop status = %d, body = %s", stopped.Code, stopped.Body.String())
	}
	if manager.Active() != 0 {
		t.Fatalf("active sessions after stop = %d", manager.Active())
	}

	readyAgain := httptest.NewRecorder()
	handler.ServeHTTP(readyAgain, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil))
	if readyAgain.Code != http.StatusOK {
		t.Fatalf("readiness after stop = %d", readyAgain.Code)
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Drain(drainCtx); err != nil {
		t.Fatalf("Drain() error = %v", err)
	}
}

func TestHandlerExposesMediaMetrics(t *testing.T) {
	cfg := validRuntimeConfig()
	manager, err := session.NewManager(1, cfg.AttachTimeout, map[session.Engine]session.Starter{
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

	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	sessionConfig := session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: session.EngineEcho, InputFormat: format, OutputFormat: format,
	}
	if err := manager.Start(context.Background(), sessionConfig); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"leamout_media_sessions_active 1",
		"leamout_media_sessions_started_total 1",
		"leamout_media_attach_latency_count",
		"leamout_media_turn_latency_count",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q: %s", want, body)
		}
	}
}
