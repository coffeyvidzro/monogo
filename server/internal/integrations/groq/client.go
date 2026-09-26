package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

func (c *Client) Generate(ctx context.Context, cfg Config, messages []Message) (Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("groq context is required")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("groq API key is required")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("groq messages are required")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("groq endpoint must be an absolute HTTPS URL")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}
	payload, err := json.Marshal(CompletionRequest{
		Model:               model,
		Messages:            messages,
		Stream:              true,
		Temperature:         cfg.Temperature,
		MaxCompletionTokens: cfg.MaxCompletionTokens,
		Tools:               cfg.Tools,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Groq request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Groq request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("start Groq stream: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		defer func() {
			_ = response.Body.Close()
		}()
		body, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
		return nil, fmt.Errorf("start Groq stream: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	result := newStream(ctx, response.Body)
	go result.readLoop()
	return result, nil
}
