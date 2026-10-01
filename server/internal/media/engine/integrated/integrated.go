// Package integrated implements end-to-end realtime voice engines.
package integrated

import (
	"context"
	"encoding/json"
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
	if runtime, ok := cfg.Provider("realtime"); ok {
		if runtime.Provider != "openai" {
			return nil, fmt.Errorf("unsupported integrated realtime provider %q", runtime.Provider)
		}
		providerConfig.APIKey = runtime.APIKey
		if len(runtime.Config) != 0 {
			var options struct {
				Endpoint         string `json:"endpoint"`
				Model            string `json:"model"`
				Voice            string `json:"voice"`
				SafetyIdentifier string `json:"safety_identifier"`
			}
			if err := json.Unmarshal(runtime.Config, &options); err != nil {
				return nil, fmt.Errorf("decode OpenAI provider config: %w", err)
			}
			if value := strings.TrimSpace(options.Endpoint); value != "" {
				providerConfig.Endpoint = value
			}
			if value := strings.TrimSpace(options.Model); value != "" {
				providerConfig.Model = value
			}
			if value := strings.TrimSpace(options.Voice); value != "" {
				providerConfig.Voice = value
			}
			if value := strings.TrimSpace(options.SafetyIdentifier); value != "" {
				providerConfig.SafetyIdentifier = value
			}
		}
	}
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
