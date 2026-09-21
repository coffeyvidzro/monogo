// Package managed owns the upstream clients used by Leamout Managed Carrier.
// DIDWW supplies DID inventory, acquisition, and inbound routing, while
// CommPeak supplies SIP origination, termination, and usage records.
package managed

import (
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/commpeak"
	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
)

type Config struct {
	DIDWWAPIKey           string
	DIDWWBaseURL          string
	CommPeakAuthorization string
	CommPeakBaseURL       string
}

// Providers contains only platform-owned carrier clients. An empty credential
// disables that provider so BYOC-only deployments do not require managed
// carrier secrets.
type Providers struct {
	DIDWW    *didww.Client
	CommPeak *commpeak.Client
}

func New(cfg Config) (*Providers, error) {
	providers := &Providers{}
	var err error
	if cfg.DIDWWAPIKey != "" {
		providers.DIDWW, err = didww.New(didww.Config{
			APIKey:  cfg.DIDWWAPIKey,
			BaseURL: cfg.DIDWWBaseURL,
		})
		if err != nil {
			return nil, fmt.Errorf("initialize DIDWW managed-number provider: %w", err)
		}
	}
	if cfg.CommPeakAuthorization != "" {
		providers.CommPeak, err = commpeak.New(commpeak.Config{
			Authorization: cfg.CommPeakAuthorization,
			BaseURL:       cfg.CommPeakBaseURL,
		})
		if err != nil {
			return nil, fmt.Errorf("initialize CommPeak managed-voice provider: %w", err)
		}
	}
	return providers, nil
}

func (p *Providers) DIDWWConfigured() bool {
	return p != nil && p.DIDWW != nil
}

func (p *Providers) CommPeakConfigured() bool {
	return p != nil && p.CommPeak != nil
}
