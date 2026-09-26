package cartesia

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
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

func (c *Client) Synthesize(ctx context.Context, cfg Config, text string, format session.AudioFormat) (Stream, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("cartesia transcript is required")
	}
	result, err := c.StartSynthesis(ctx, cfg, format)
	if err != nil {
		return nil, err
	}
	if err := result.SendText(ctx, text, false); err != nil {
		_ = result.Close()
		return nil, fmt.Errorf("start cartesia synthesis: %w", err)
	}
	return result, nil
}

func (c *Client) StartSynthesis(ctx context.Context, cfg Config, format session.AudioFormat) (TextStream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cartesia context is required")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("cartesia API key is required")
	}
	if strings.TrimSpace(cfg.VoiceID) == "" {
		return nil, fmt.Errorf("cartesia voice ID is required")
	}
	if err := format.Validate(); err != nil {
		return nil, fmt.Errorf("cartesia audio format: %w", err)
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "wss" || parsed.Host == "" {
		return nil, fmt.Errorf("cartesia endpoint must be an absolute wss URL")
	}
	version := strings.TrimSpace(cfg.APIVersion)
	if version == "" {
		version = DefaultAPIVersion
	}
	query := parsed.Query()
	query.Set("cartesia_version", version)
	parsed.RawQuery = query.Encode()
	header := http.Header{
		"X-API-Key": []string{strings.TrimSpace(cfg.APIKey)},
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
			return nil, fmt.Errorf("connect cartesia: HTTP %d: %w", response.StatusCode, err)
		}
		return nil, fmt.Errorf("connect cartesia: %w", err)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}
	streamCtx, cancel := context.WithCancel(ctx)
	result := newStream(streamCtx, cancel, connection, format, uuid.NewString())
	result.request = GenerationRequest{
		ModelID:   model,
		Voice:     strings.TrimSpace(cfg.VoiceID),
		Language:  strings.TrimSpace(cfg.Language),
		ContextID: result.contextID,
		OutputFormat: OutputFormat{
			Container:  "raw",
			Encoding:   "pcm_s16le",
			SampleRate: format.SampleRateHz,
		},
	}
	go result.readLoop()
	return result, nil
}
