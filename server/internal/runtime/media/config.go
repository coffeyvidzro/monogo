package media

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddress    string        `env:"MEDIA_LISTEN_ADDRESS" envDefault:":8090"`
	PublicWebSocket  string        `env:"MEDIA_PUBLIC_WS_URL" envDefault:"ws://media:8090/v1/audio-forks"`
	TokenSecret      string        `env:"MEDIA_TOKEN_SECRET,required"`
	ControlToken     string        `env:"MEDIA_CONTROL_TOKEN,required"`
	TokenTTL         time.Duration `env:"MEDIA_TOKEN_TTL" envDefault:"30s"`
	MaxSessions      int           `env:"MEDIA_MAX_SESSIONS" envDefault:"100"`
	ReadLimit        int64         `env:"MEDIA_MAX_FRAME_BYTES" envDefault:"65536"`
	HandshakeTimeout time.Duration `env:"MEDIA_HANDSHAKE_TIMEOUT" envDefault:"5s"`
	DrainTimeout     time.Duration `env:"MEDIA_DRAIN_TIMEOUT" envDefault:"30s"`
	OpenAIAPIKey     string        `env:"OPENAI_API_KEY"`
	DeepgramAPIKey   string        `env:"DEEPGRAM_API_KEY"`
	GroqAPIKey       string        `env:"GROQ_API_KEY"`
	CartesiaAPIKey   string        `env:"CARTESIA_API_KEY"`
	CartesiaVoiceID  string        `env:"CARTESIA_VOICE_ID"`
}

func loadConfig() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse media environment: %w", err)
	}
	cfg.ListenAddress = strings.TrimSpace(cfg.ListenAddress)
	cfg.PublicWebSocket = strings.TrimRight(strings.TrimSpace(cfg.PublicWebSocket), "/")
	cfg.TokenSecret = strings.TrimSpace(cfg.TokenSecret)
	cfg.ControlToken = strings.TrimSpace(cfg.ControlToken)
	cfg.OpenAIAPIKey = strings.TrimSpace(cfg.OpenAIAPIKey)
	cfg.DeepgramAPIKey = strings.TrimSpace(cfg.DeepgramAPIKey)
	cfg.GroqAPIKey = strings.TrimSpace(cfg.GroqAPIKey)
	cfg.CartesiaAPIKey = strings.TrimSpace(cfg.CartesiaAPIKey)
	cfg.CartesiaVoiceID = strings.TrimSpace(cfg.CartesiaVoiceID)
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
	if len(c.TokenSecret) < 32 {
		return fmt.Errorf("media token secret must contain at least 32 bytes")
	}
	if len(c.ControlToken) < 32 {
		return fmt.Errorf("media control token must contain at least 32 bytes")
	}
	if c.TokenTTL <= 0 || c.HandshakeTimeout <= 0 || c.DrainTimeout <= 0 {
		return fmt.Errorf("media timeouts must be positive")
	}
	if c.MaxSessions <= 0 || c.ReadLimit <= 0 {
		return fmt.Errorf("media limits must be positive")
	}
	return nil
}
