package media

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"strings"
	"sync/atomic"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/media/transport"
	"github.com/google/uuid"
)

type handler struct {
	config    Config
	manager   *session.Manager
	tokens    *transport.TokenService
	websocket http.Handler
	draining  atomic.Bool
}

func newHandler(cfg Config, manager *session.Manager, tokens *transport.TokenService, websocketHandler http.Handler) *handler {
	return &handler{config: cfg, manager: manager, tokens: tokens, websocket: websocketHandler}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/livez":
		writeText(w, http.StatusOK, "live\n")
	case r.Method == http.MethodGet && r.URL.Path == "/readyz":
		if h.draining.Load() || !h.manager.Ready() {
			writeText(w, http.StatusServiceUnavailable, "not ready\n")
			return
		}
		writeText(w, http.StatusOK, "ready\n")
	case r.URL.Path == "/v1/audio-forks":
		h.websocket.ServeHTTP(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/internal/v1/sessions":
		h.createSession(w, r)
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/internal/v1/sessions/") &&
		strings.HasSuffix(r.URL.Path, "/control"):
		h.controlSession(w, r)
	case r.Method == http.MethodDelete &&
		strings.HasPrefix(r.URL.Path, "/internal/v1/sessions/"):
		h.stopSession(w, r)
	default:
		http.NotFound(w, r)
	}
}

type createSessionResponse struct {
	WebSocketURL        string `json:"websocket_url"`
	ControlWebSocketURL string `json:"control_websocket_url"`
}

func (h *handler) createSession(w http.ResponseWriter, r *http.Request) {
	if h.draining.Load() {
		http.Error(w, "media worker is draining", http.StatusServiceUnavailable)
		return
	}
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer func() {
		_ = r.Body.Close()
	}()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var cfg session.Config
	if err := decoder.Decode(&cfg); err != nil {
		http.Error(w, "invalid session configuration", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		http.Error(w, "invalid session configuration", http.StatusBadRequest)
		return
	}
	if err := h.manager.Start(r.Context(), cfg); err != nil {
		if !errors.Is(err, session.ErrSessionAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		existing, ok := h.manager.Config(cfg.ID)
		if !ok || !existing.Equal(cfg) {
			http.Error(w, "media session id conflicts with another configuration", http.StatusConflict)
			return
		}
	}
	token, err := h.tokens.Issue(transport.TokenClaims{
		SessionID: cfg.ID, CallID: cfg.CallID, ChannelID: cfg.ChannelID, OrganizationID: cfg.OrganizationID,
	}, h.config.TokenTTL)
	if err != nil {
		_ = h.manager.Stop(r.Context(), cfg.ID)
		http.Error(w, "issue media token", http.StatusInternalServerError)
		return
	}
	websocketURL, err := url.Parse(h.config.PublicWebSocket)
	if err != nil {
		_ = h.manager.Stop(r.Context(), cfg.ID)
		http.Error(w, "build media URL", http.StatusInternalServerError)
		return
	}
	query := websocketURL.Query()
	query.Set("token", token)
	websocketURL.RawQuery = query.Encode()
	controlURL, err := h.controlWebSocketURL(r, cfg.ID)
	if err != nil {
		_ = h.manager.Stop(r.Context(), cfg.ID)
		http.Error(w, "build media control URL", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createSessionResponse{
		WebSocketURL: websocketURL.String(),
		ControlWebSocketURL: controlURL,
	})
}

func (h *handler) controlSession(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rawID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/internal/v1/sessions/"), "/control")
	if rawID == "" || strings.Contains(rawID, "/") {
		http.Error(w, "invalid media session id", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid media session id", http.StatusBadRequest)
		return
	}
	attachment, err := h.manager.AttachControl(id)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrSessionNotFound):
			http.Error(w, "media session not found", http.StatusNotFound)
		case errors.Is(err, session.ErrControlAlreadyAttached):
			http.Error(w, "media control already attached", http.StatusConflict)
		default:
			http.Error(w, "attach media control", http.StatusInternalServerError)
		}
		return
	}
	defer attachment.Close()

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return
	}
	defer func() { _ = ws.CloseNow() }()

	ctx := r.Context()
	errs := make(chan error, 2)
	go func() {
		for event := range attachment.Events() {
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := ws.Write(writeCtx, websocket.MessageText, mustJSON(event))
			cancel()
			if err != nil {
				errs <- err
				return
			}
		}
		errs <- nil
	}()
	go func() {
		for {
			kind, payload, err := ws.Read(ctx)
			if err != nil {
				errs <- err
				return
			}
			if kind != websocket.MessageText {
				errs <- fmt.Errorf("media control requires text messages")
				return
			}
			var command session.Command
			if err := json.Unmarshal(payload, &command); err != nil {
				errs <- fmt.Errorf("decode media command: %w", err)
				return
			}
			if err := attachment.Command(ctx, command); err != nil {
				errs <- err
				return
			}
			if command.Type == session.CommandStop {
				errs <- nil
				return
			}
		}
	}()
	<-errs
}

func (h *handler) controlWebSocketURL(r *http.Request, id uuid.UUID) (string, error) {
	scheme := "ws"
	if r.TLS != nil {
		scheme = "wss"
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return "", fmt.Errorf("media control host is required")
	}
	return (&url.URL{
		Scheme: scheme,
		Host: host,
		Path: "/internal/v1/sessions/" + id.String() + "/control",
	}).String(), nil
}

func mustJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}

func (h *handler) stopSession(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rawID := strings.TrimPrefix(r.URL.Path, "/internal/v1/sessions/")
	if rawID == "" || strings.Contains(rawID, "/") {
		http.Error(w, "invalid media session id", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid media session id", http.StatusBadRequest)
		return
	}
	if err := h.manager.Stop(r.Context(), id); err != nil &&
		!errors.Is(err, session.ErrSessionNotFound) {
		http.Error(w, "stop media session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) authorized(r *http.Request) bool {
	want := []byte("Bearer " + h.config.ControlToken)
	got := []byte(strings.TrimSpace(r.Header.Get("Authorization")))
	return len(got) == len(want) && subtle.ConstantTimeCompare(got, want) == 1
}

func writeText(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprint(w, value)
}
