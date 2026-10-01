package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

type Provider struct {
	Client *Client
	Config Config
}

func (p Provider) Descriptor() providersdk.Descriptor {
	return providersdk.Descriptor{
		ID:   "openai",
		Kind: providersdk.KindRealtime,
		Capabilities: []providersdk.Capability{
			providersdk.CapabilityStreaming,
			providersdk.CapabilityTurnDetection,
			providersdk.CapabilityToolCalling,
			providersdk.CapabilityUsage,
			providersdk.CapabilityBargeIn,
		},
	}
}

func (p Provider) StartRealtime(
	ctx context.Context,
	runtime providersdk.Runtime,
	cfg session.Config,
) (session.Stream, error) {
	providerConfig := p.Config
	if strings.TrimSpace(runtime.APIKey) != "" {
		providerConfig.APIKey = runtime.APIKey
	}
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
	if value := strings.TrimSpace(cfg.Voice); value != "" {
		providerConfig.Voice = value
	}

	providerConfig.Tools = make([]Tool, 0, len(cfg.Tools))
	for _, tool := range cfg.Tools {
		providerConfig.Tools = append(providerConfig.Tools, Tool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	client := p.Client
	if client == nil {
		client = NewClient(nil)
	}
	return client.Start(ctx, providerConfig, cfg)
}

var _ providersdk.Realtime = Provider{}
