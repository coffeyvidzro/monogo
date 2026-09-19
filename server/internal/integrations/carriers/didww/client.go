package didww

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

const DefaultBaseURL = "https://api.didww.com/v3"
const maxResponseBytes = 4 << 20

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("didww API returned status %d", e.StatusCode)
}

func New(cfg Config) (*Client, error) {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	if cfg.APIKey == "" || strings.ContainsAny(cfg.APIKey, "\r\n") {
		return nil, fmt.Errorf("didww API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"))
	if err != nil || base == nil || (base.Scheme != "https" && base.Scheme != "http") ||
		base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("didww base URL is invalid")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    base.String(),
		httpClient: cfg.HTTPClient,
	}, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, target any) error {
	if ctx == nil {
		return fmt.Errorf("didww context is required")
	}
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("build didww request: %w", err)
	}
	endpoint.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create didww request: %w", err)
	}
	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/vnd.api+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send didww request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &APIError{StatusCode: resp.StatusCode}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read didww response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return fmt.Errorf("didww response exceeds size limit")
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode didww response: %w", err)
	}
	return nil
}
