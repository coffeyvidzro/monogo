package deepgram

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
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

func (c *Client) Start(ctx context.Context, cfg Config, format session.AudioFormat) (Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("deepgram context is required")
	}
	if err := format.Validate(); err != nil {
		return nil, fmt.Errorf("deepgram audio format: %w", err)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("deepgram API key is required")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "wss" || parsed.Host == "" {
		return nil, fmt.Errorf("deepgram endpoint must be an absolute wss URL")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}
	query := parsed.Query()
	query.Set("model", model)
	query.Set("encoding", "linear16")
	query.Set("sample_rate", strconv.Itoa(format.SampleRateHz))
	query.Set("channels", strconv.Itoa(format.Channels))
	query.Set("interim_results", strconv.FormatBool(cfg.InterimResults))
	if language := strings.TrimSpace(cfg.Language); language != "" {
		query.Set("language", language)
	}
	if cfg.Endpointing > 0 {
		query.Set("endpointing", strconv.FormatInt(cfg.Endpointing.Milliseconds(), 10))
	}
	parsed.RawQuery = query.Encode()
	header := http.Header{
		"Authorization": []string{"Token " + strings.TrimSpace(cfg.APIKey)},
	}
	connection, response, err := websocket.Dial(ctx, parsed.String(), &websocket.DialOptions{
		HTTPClient:      c.httpClient,
		HTTPHeader:      header,
		CompressionMode: websocket.CompressionDisabled,
	})
	if response != nil && response.Body != nil {
		defer func() {
			_ = response.Body.Close()
		}()
	}
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("connect Deepgram: HTTP %d: %w", response.StatusCode, err)
		}
		return nil, fmt.Errorf("connect Deepgram: %w", err)
	}
	streamCtx, cancel := context.WithCancel(ctx)
	result := newStream(streamCtx, cancel, connection, format)
	go result.readLoop()
	return result, nil
}
