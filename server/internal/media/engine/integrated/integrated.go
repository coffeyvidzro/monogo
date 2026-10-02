// Package integrated implements end-to-end realtime voice engines.
package integrated

import (
	"context"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

type Engine struct {
	Registry        *providersdk.Registry
	DefaultProvider string
}

func (e Engine) Start(ctx context.Context, cfg session.Config) (session.Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("integrated engine context is required")
	}
	if cfg.Engine != session.EngineIntegrated {
		return nil, fmt.Errorf("integrated engine cannot start session engine %q", cfg.Engine)
	}
	if e.Registry == nil {
		return nil, fmt.Errorf("provider registry is required")
	}

	providerID := strings.TrimSpace(e.DefaultProvider)
	if providerID == "" {
		providerID = "openai"
	}
	runtime := providersdk.Runtime{}
	if configured, ok := cfg.Provider("realtime"); ok {
		providerID = strings.TrimSpace(configured.Provider)
		runtime = providersdk.Runtime{
			APIKey: configured.APIKey,
			Config: append([]byte(nil), configured.Config...),
		}
	}

	provider, ok := e.Registry.Realtime(providerID)
	if !ok {
		return nil, fmt.Errorf("realtime provider %q is not registered", providerID)
	}
	return provider.StartRealtime(ctx, runtime, cfg)
}
