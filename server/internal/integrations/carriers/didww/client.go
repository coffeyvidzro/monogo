package didww

import (
	"bytes"
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
const DefaultAPIVersion = "2026-04-16"
const jsonAPIMediaType = "application/vnd.api+json"
const maxResponseBytes = 4 << 20

type Client struct {
	apiKey     string
	apiVersion string
	baseURL    string
	httpClient *http.Client
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
	if err != nil || base == nil || base.Host == "" || base.User != nil ||
		base.RawQuery != "" || base.Fragment != "" ||
		(base.Scheme != "https" && !(base.Scheme == "http" && (base.Hostname() == "localhost" || base.Hostname() == "127.0.0.1" || base.Hostname() == "::1"))) {
		return nil, fmt.Errorf("didww base URL must be HTTPS (HTTP permitted for localhost tests)")
	}
	if !strings.HasSuffix(base.Path, "/v3") {
		return nil, fmt.Errorf("didww base URL must end with /v3")
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = DefaultAPIVersion
	}
	if strings.TrimSpace(cfg.APIVersion) != DefaultAPIVersion {
		return nil, fmt.Errorf("didww API version must match the supported 2026-04-16 contract")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	// A redirect must never forward the platform's DIDWW API key to another host.
	transport := *cfg.HTTPClient
	transport.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{
		apiKey:     cfg.APIKey,
		apiVersion: DefaultAPIVersion,
		baseURL:    base.String(),
		httpClient: &transport,
	}, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, target any) error {
	return c.do(ctx, http.MethodGet, path, params, nil, target)
}

func (c *Client) do(ctx context.Context, method, path string, params url.Values, body any, target any) error {
	if ctx == nil {
		return fmt.Errorf("didww context is required")
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "?#") {
		return fmt.Errorf("didww request path is invalid")
	}
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("build didww request: %w", err)
	}
	endpoint.RawQuery = params.Encode()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode didww request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return fmt.Errorf("create didww request: %w", err)
	}
	req.Header.Set("api-key", c.apiKey)
	req.Header.Set("X-DIDWW-Api-Version", c.apiVersion)
	req.Header.Set("Accept", jsonAPIMediaType)
	if body != nil {
		req.Header.Set("Content-Type", jsonAPIMediaType)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send didww request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(resp)
	}
	if target == nil {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read didww response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return fmt.Errorf("didww response exceeds size limit")
	}
	if len(payload) == 0 || string(payload) == "null" {
		return fmt.Errorf("didww response is empty")
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode didww response: %w", err)
	}
	return nil
}
