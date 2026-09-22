package coturn

import (
	"fmt"
	"net/url"
	"strings"
)

// Config contains the credential-issuer settings shared with the Coturn
// container. URLs are publicly returned as ICE server configuration.
type Config struct {
	AuthSecret string
	URLs       []string
}

func (c Config) validate() error {
	if len(c.AuthSecret) < 32 {
		return fmt.Errorf("TURN authentication secret must be at least 32 bytes")
	}
	if len(c.URLs) == 0 {
		return fmt.Errorf("at least one TURN or STUN URL is required")
	}
	for _, rawURL := range c.URLs {
		if !validICEURL(rawURL) {
			return fmt.Errorf("invalid TURN or STUN URL %q", rawURL)
		}
	}
	return nil
}

func validICEURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Fragment != "" {
		return false
	}
	switch parsed.Scheme {
	case "stun", "stuns", "turn", "turns":
	default:
		return false
	}

	// RFC 7064/7065 URLs normally parse the host and port into Opaque
	// because they do not include // after the scheme.
	target := parsed.Opaque
	if target == "" {
		target = parsed.Host
	}
	target, _, _ = strings.Cut(target, "?")
	return strings.TrimSpace(target) != ""
}
