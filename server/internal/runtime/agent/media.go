package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

const maxMediaControlResponse = 64 << 10

type mediaClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type createMediaSessionResponse struct {
	WebSocketURL        string `json:"websocket_url"`
	ControlWebSocketURL string `json:"control_websocket_url"`
}

type mediaSessionEndpoints struct {
	AudioURL   string
	ControlURL string
}

func newMediaClient(cfg Config) (*mediaClient, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &mediaClient{
		baseURL: strings.TrimRight(cfg.MediaControlURL, "/"),
		token:   cfg.MediaControlToken,
		client:  &http.Client{Timeout: cfg.RequestTimeout},
	}, nil
}

func (c *mediaClient) CreateSession(ctx context.Context, cfg session.Config) (mediaSessionEndpoints, error) {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("marshal media session: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/sessions", bytes.NewReader(payload))
	if err != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("create media session request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("create media session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxMediaControlResponse))
	if readErr != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("read media session response: %w", readErr)
	}
	if resp.StatusCode != http.StatusCreated {
		return mediaSessionEndpoints{}, fmt.Errorf("create media session: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result createMediaSessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("decode media session response: %w", err)
	}
	audioURL, err := parseMediaWebSocketURL(result.WebSocketURL)
	if err != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("media session returned an invalid audio WebSocket URL")
	}
	controlURL, err := parseMediaWebSocketURL(result.ControlWebSocketURL)
	if err != nil {
		return mediaSessionEndpoints{}, fmt.Errorf("media session returned an invalid control WebSocket URL")
	}
	return mediaSessionEndpoints{AudioURL: audioURL, ControlURL: controlURL}, nil
}

func (c *mediaClient) StopSession(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("media session id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/internal/v1/sessions/"+id.String(), nil)
	if err != nil {
		return fmt.Errorf("create media stop request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("stop media session: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxMediaControlResponse))
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("stop media session: HTTP %d", resp.StatusCode)
	}
	return nil
}

func parseMediaWebSocketURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Host == "" {
		return "", fmt.Errorf("invalid media WebSocket URL")
	}
	return parsed.String(), nil
}
