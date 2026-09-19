package commpeak

import (
	"fmt"
	"net/netip"
	"strings"
)

// Validate checks the configured SIP host/port/transport without guessing
// CommPeak endpoints, dial prefixes or authentication requirements.
func (e SIPEndpoint) Validate() error {
	host := e.Host
	if host == "" || len(host) > 253 || strings.TrimSpace(host) != host ||
		strings.ContainsAny(host, "/\\@?# \t\r\n") {
		return fmt.Errorf("commpeak SIP host is invalid")
	}
	if _, err := netip.ParseAddr(host); err != nil {
		for _, label := range strings.Split(host, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return fmt.Errorf("commpeak SIP host is invalid")
			}
			for _, ch := range label {
				if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') &&
					(ch < '0' || ch > '9') && ch != '-' {
					return fmt.Errorf("commpeak SIP host is invalid")
				}
			}
		}
	}
	if e.Port == 0 {
		return fmt.Errorf("commpeak SIP port is required")
	}
	switch e.Transport {
	case "udp", "tcp", "tls":
		return nil
	default:
		return fmt.Errorf("commpeak SIP transport must be udp, tcp or tls")
	}
}

// Validate verifies an already-provisioned outbound trunk. Credentials are
// optional because SIP IP authentication may be configured at the carrier.
func (cfg TerminationConfig) Validate() error {
	if err := cfg.Endpoint.Validate(); err != nil {
		return err
	}
	if cfg.Credentials != nil {
		username := cfg.Credentials.Username
		if username == "" || strings.TrimSpace(username) != username ||
			strings.ContainsAny(username, "\r\n") || cfg.Credentials.Password == "" {
			return fmt.Errorf("commpeak SIP credentials are invalid")
		}
	}
	return nil
}

// SIPDestination provides the validated termination endpoint to Leamout's
// routing/runtime layers. No provider SIP request or call is sent here.
func (cfg TerminationConfig) SIPDestination() (SIPEndpoint, error) {
	if err := cfg.Validate(); err != nil {
		return SIPEndpoint{}, err
	}
	return cfg.Endpoint, nil
}
