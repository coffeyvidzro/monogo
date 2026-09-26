// Package integrated implements end-to-end realtime voice engines.
package integrated

import (
	"context"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Engine struct {
	Client *openai.Client
	Config openai.Config
}

func (e Engine) Start(ctx context.Context, cfg session.Config) (session.Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("integrated engine context is required")
	}
	if cfg.Engine != session.EngineIntegrated {
		return nil, fmt.Errorf("integrated engine cannot start session engine %q", cfg.Engine)
	}
	client := e.Client
	if client == nil {
		client = openai.NewClient(nil)
	}
	providerConfig := e.Config
	if voice := strings.TrimSpace(cfg.Voice); voice != "" {
		providerConfig.Voice = voice
	}
	return client.Start(ctx, providerConfig, cfg)
}
