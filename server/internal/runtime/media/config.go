package media

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	RedisURL         string        `env:"REDIS_URL"`
	NodeID           string        `env:"MEDIA_NODE_ID"`
	ControlURL       string        `env:"MEDIA_CONTROL_URL"`
	ListenAddress    string        `env:"MEDIA_LISTEN_ADDRESS" envDefault:":8090"`
	PublicWebSocket  string        `env:"MEDIA_PUBLIC_WS_URL" envDefault:"ws://media:8090/v1/audio-forks"`
	TokenSecret      string        `env:"MEDIA_TOKEN_SECRET,required"`
	ControlToken     string        `env:"MEDIA_CONTROL_TOKEN,required"`
	TokenTTL         time.Duration `env:"MEDIA_TOKEN_TTL" envDefault:"30s"`
	AttachTimeout    time.Duration `env:"MEDIA_ATTACH_TIMEOUT" envDefault:"30s"`
	MaxSessions      int           `env:"MEDIA_MAX_SESSIONS" envDefault:"100"`
	ReadLimit        int64         `env:"MEDIA_MAX_FRAME_BYTES" envDefault:"65536"`
	HandshakeTimeout time.Duration `env:"MEDIA_HANDSHAKE_TIMEOUT" envDefault:"5s"`
	DrainTimeout     time.Duration `env:"MEDIA_DRAIN_TIMEOUT" envDefault:"30s"`
	OpenAIAPIKey     string        `env:"OPENAI_API_KEY"`
	OpenAIEndpoint   string        `env:"OPENAI_REALTIME_ENDPOINT"`
}

func loadConfig() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse media environment: %w", err)
	}
	cfg.RedisURL = strings.TrimSpace(cfg.RedisURL)
	cfg.NodeID = strings.TrimSpace(cfg.NodeID)
	cfg.ControlURL = strings.TrimRight(strings.TrimSpace(cfg.ControlURL), "/")
	cfg.ListenAddress = strings.TrimSpace(cfg.ListenAddress)
	cfg.PublicWebSocket = strings.TrimRight(strings.TrimSpace(cfg.PublicWebSocket), "/")
	if cfg.NodeID == "" {
		cfg.NodeID, _ = os.Hostname()
	}
	if cfg.ControlURL == "" {
		if publicURL, parseErr := url.Parse(cfg.PublicWebSocket); parseErr == nil && publicURL.Host != "" {
			scheme := "http"
			if publicURL.Scheme == "wss" {
				scheme = "https"
			}
			cfg.ControlURL = scheme + "://" + publicURL.Host
		}
	}
	cfg.TokenSecret = strings.TrimSpace(cfg.TokenSecret)
	cfg.ControlToken = strings.TrimSpace(cfg.ControlToken)
	cfg.OpenAIAPIKey = strings.TrimSpace(cfg.OpenAIAPIKey)
	cfg.OpenAIEndpoint = strings.TrimSpace(cfg.OpenAIEndpoint)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ListenAddress) == "" {
		return fmt.Errorf("media listen address is required")
	}
	parsed, err := url.Parse(c.PublicWebSocket)
	if err != nil || (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Host == "" {
		return fmt.Errorf("media public WebSocket URL must be an absolute ws or wss URL")
	}
	if c.RedisURL != "" {
		control, controlErr := url.Parse(c.ControlURL)
		if controlErr != nil || (control.Scheme != "http" && control.Scheme != "https") || control.Host == "" {
			return fmt.Errorf("media control URL must be an absolute http or https URL")
		}
		if strings.TrimSpace(c.NodeID) == "" {
			return fmt.Errorf("media node id is required when Redis registration is enabled")
		}
	}
	if len(c.TokenSecret) < 32 {
		return fmt.Errorf("media token secret must contain at least 32 bytes")
	}
	if len(c.ControlToken) < 32 {
		return fmt.Errorf("media control token must contain at least 32 bytes")
	}
	if c.OpenAIEndpoint != "" {
		endpoint, endpointErr := url.Parse(c.OpenAIEndpoint)
		if endpointErr != nil || endpoint.Scheme != "wss" || endpoint.Host == "" {
			return fmt.Errorf("OpenAI Realtime endpoint must be an absolute wss URL")
		}
	}
	if c.TokenTTL <= 0 || c.AttachTimeout <= 0 || c.HandshakeTimeout <= 0 || c.DrainTimeout <= 0 {
		return fmt.Errorf("media timeouts must be positive")
	}
	if c.MaxSessions <= 0 || c.ReadLimit <= 0 {
		return fmt.Errorf("media limits must be positive")
	}
	return nil
}
