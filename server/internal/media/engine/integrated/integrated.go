// Package integrated implements end-to-end realtime voice engines.
package integrated

import (
	"context"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

const providerSampleRateHz = 24000

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
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	client := e.Client
	if client == nil {
		client = openai.NewClient(nil)
	}
	providerConfig := e.Config
	if voice := strings.TrimSpace(cfg.Voice); voice != "" {
		providerConfig.Voice = voice
	}

	providerFormat := session.AudioFormat{SampleRateHz: providerSampleRateHz, Channels: 1}
	providerSession := cfg
	providerSession.InputFormat = providerFormat
	providerSession.OutputFormat = providerFormat
	providerStream, err := client.Start(ctx, providerConfig, providerSession)
	if err != nil {
		return nil, err
	}
	if cfg.InputFormat == providerFormat && cfg.OutputFormat == providerFormat {
		return providerStream, nil
	}

	adapted, err := newRateAdaptedStream(ctx, providerStream, cfg.InputFormat, cfg.OutputFormat, providerFormat)
	if err != nil {
		_ = providerStream.Close(context.Background())
		return nil, err
	}
	return adapted, nil
}
