package coturn

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client is a local TURN REST credential issuer. It does not open an HTTP
// connection to the Coturn daemon.
type Client struct {
	authSecret string
	urls       []string
}

// New validates the configuration and copies the ICE URLs so callers cannot
// mutate the response configuration after initialization.
func New(config Config) (*Client, error) {
	config.AuthSecret = strings.TrimSpace(config.AuthSecret)
	urls := make([]string, len(config.URLs))
	for i, rawURL := range config.URLs {
		urls[i] = strings.TrimSpace(rawURL)
	}
	config.URLs = urls
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &Client{
		authSecret: config.AuthSecret,
		urls:       append([]string(nil), config.URLs...),
	}, nil
}

// URLs returns an independent copy of the configured public ICE endpoints.
func (c *Client) URLs() []string {
	return append([]string(nil), c.urls...)
}

// Issue creates a short-lived TURN REST credential. Tests may pass an explicit
// clock and nonce source; nil nonceSource uses the system CSPRNG.
func (c *Client) Issue(organizationID uuid.UUID, now time.Time, nonceSource io.Reader) (Credential, error) {
	if c == nil {
		return Credential{}, fmt.Errorf("coturn client is required")
	}
	if organizationID == uuid.Nil {
		return Credential{}, fmt.Errorf("organization id is required")
	}
	if nonceSource == nil {
		nonceSource = rand.Reader
	}
	return c.issue(organizationID, now, nonceSource)
}
