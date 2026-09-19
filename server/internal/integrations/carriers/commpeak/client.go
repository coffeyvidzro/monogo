package commpeak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.commpeak.com"

const maxResponseBytes = 4 << 20

// Client is a control-plane HTTP client, not a SIP call controller.
type Client struct {
	authorization string
	baseURL       string
	httpClient    *http.Client
}

// APIError reports an upstream HTTP status without exposing credentials or CDR data.
type APIError struct {
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("commpeak API returned status %d", e.StatusCode)
}

func New(cfg Config) (*Client, error) {
	cfg.Authorization = strings.TrimSpace(cfg.Authorization)
	if cfg.Authorization == "" || strings.ContainsAny(cfg.Authorization, "\r\n") {
		return nil, fmt.Errorf("commpeak authorization is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"))
	if err != nil || base == nil || (base.Scheme != "https" && base.Scheme != "http") ||
		base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("commpeak base URL is invalid")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		authorization: cfg.Authorization,
		baseURL:       base.String(),
		httpClient:    cfg.HTTPClient,
	}, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	if ctx == nil {
		return nil, fmt.Errorf("commpeak context is required")
	}
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("build commpeak request: %w", err)
	}
	endpoint.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create commpeak request: %w", err)
	}
	req.Header.Set("Authorization", c.authorization)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send commpeak request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &APIError{StatusCode: resp.StatusCode}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read commpeak response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return nil, fmt.Errorf("commpeak response exceeds size limit")
	}
	if !json.Valid(payload) || string(payload) == "null" {
		return nil, fmt.Errorf("commpeak response is not valid JSON")
	}
	return json.RawMessage(payload), nil
}
