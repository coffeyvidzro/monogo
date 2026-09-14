package opensips

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("OpenSIPS context is required")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("OpenSIPS URL is required")
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 5 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: cfg.ConnectTimeout}).DialContext

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.RequestTimeout,
		},
		baseURL: baseURL,
	}, nil
}

func (c *Client) Command(ctx context.Context, command Command) (Response, error) {
	if err := c.validate(ctx); err != nil {
		return Response{}, err
	}
	if err := command.Validate(); err != nil {
		return Response{}, err
	}

	payload, err := json.Marshal(command)
	if err != nil {
		return Response{}, fmt.Errorf("encode OpenSIPS command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return Response{}, fmt.Errorf("create OpenSIPS request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("execute OpenSIPS command: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Response{}, fmt.Errorf("OpenSIPS command returned HTTP %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{}, fmt.Errorf("decode OpenSIPS JSON-RPC response: %w", err)
	}
	if err := result.Validate(); err != nil {
		return Response{}, err
	}

	return result, nil
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) validate(ctx context.Context) error {
	if c == nil || c.httpClient == nil || c.baseURL == "" {
		return fmt.Errorf("OpenSIPS client is nil")
	}
	if ctx == nil {
		return fmt.Errorf("OpenSIPS context is required")
	}
	return nil
}
