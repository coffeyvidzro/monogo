package commpeak

import (
	"fmt"
	"net/netip"
)

// Validate requires an explicit list of verified CommPeak origination sources;
// a missing or catch-all source list must not authorize incoming SIP traffic.
func (cfg OriginationConfig) Validate() error {
	if err := cfg.Ingress.Validate(); err != nil {
		return err
	}
	if len(cfg.Sources) == 0 {
		return fmt.Errorf("commpeak origination source networks are required")
	}
	for _, prefix := range cfg.Sources {
		if !prefix.IsValid() || prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return fmt.Errorf("commpeak origination source network is invalid")
		}
	}
	return nil
}

// AllowsSource matches a signaling peer IP against an account-verified
// source allowlist. The SIP edge must enforce this check before call admission;
// this helper does not install firewall rules or authenticate SIP requests.
func (cfg OriginationConfig) AllowsSource(source netip.Addr) bool {
	if err := cfg.Validate(); err != nil || !source.IsValid() {
		return false
	}
	source = source.Unmap()
	for _, prefix := range cfg.Sources {
		if prefix.Contains(source) {
			return true
		}
	}
	return false
}
