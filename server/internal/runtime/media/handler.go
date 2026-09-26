package media

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/media/transport"
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
	default:
		http.NotFound(w, r)
	}
}

type createSessionResponse struct {
	WebSocketURL string `json:"websocket_url"`
}

func (h *handler) createSession(w http.ResponseWriter, r *http.Request) {
	if h.draining.Load() {
		http.Error(w, "media worker is draining", http.StatusServiceUnavailable)
		return
	}
	want := []byte("Bearer " + h.config.ControlToken)
	got := []byte(strings.TrimSpace(r.Header.Get("Authorization")))
	if len(got) != len(want) || subtle.ConstantTimeCompare(got, want) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer r.Body.Close()
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
		http.Error(w, err.Error(), http.StatusConflict)
		return
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createSessionResponse{WebSocketURL: websocketURL.String()})
}

func writeText(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprint(w, value)
}
