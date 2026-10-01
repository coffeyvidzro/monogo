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
	providerConfig.Tools = make([]openai.Tool, 0, len(cfg.Tools))
	for _, tool := range cfg.Tools {
		providerConfig.Tools = append(providerConfig.Tools, openai.Tool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	return client.Start(ctx, providerConfig, cfg)
}
