// Package voiceai attaches durable Voice Agent sessions to live calls.
package voiceai

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	MediaControlURL   string
	MediaControlToken string
	RequestTimeout    time.Duration
}

func DefaultConfig(mediaControlURL, mediaControlToken string) Config {
	return Config{
		MediaControlURL:   strings.TrimRight(strings.TrimSpace(mediaControlURL), "/"),
		MediaControlToken: strings.TrimSpace(mediaControlToken),
		RequestTimeout:    5 * time.Second,
	}
}

func (c Config) Validate() error {
	parsed, err := url.Parse(strings.TrimSpace(c.MediaControlURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("voice AI media control URL must be an absolute http or https URL")
	}
	if len(strings.TrimSpace(c.MediaControlToken)) < 32 {
		return fmt.Errorf("voice AI media control token must contain at least 32 bytes")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("voice AI media request timeout must be positive")
	}
	return nil
}
