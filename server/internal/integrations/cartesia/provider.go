package cartesia

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

type Provider struct {
	Client *Client
	Config Config
}

func (p Provider) Descriptor() providersdk.Descriptor {
	return providersdk.Descriptor{
		ID:   "cartesia",
		Kind: providersdk.KindTTS,
		Capabilities: []providersdk.Capability{
			providersdk.CapabilityStreaming,
		},
	}
}

func (p Provider) StartTTS(
	ctx context.Context,
	req providersdk.TTSRequest,
) (providersdk.TTSStream, error) {
	cfg := p.Config
	if strings.TrimSpace(req.Runtime.APIKey) != "" {
		cfg.APIKey = req.Runtime.APIKey
	}
	if len(req.Runtime.Config) != 0 {
		var options struct {
			Endpoint   string `json:"endpoint"`
			APIVersion string `json:"api_version"`
			Model      string `json:"model"`
			VoiceID    string `json:"voice_id"`
			Language   string `json:"language"`
		}
		if err := json.Unmarshal(req.Runtime.Config, &options); err != nil {
			return nil, fmt.Errorf("decode Cartesia provider config: %w", err)
		}
		if value := strings.TrimSpace(options.Endpoint); value != "" {
			cfg.Endpoint = value
		}
		if value := strings.TrimSpace(options.APIVersion); value != "" {
			cfg.APIVersion = value
		}
		if value := strings.TrimSpace(options.Model); value != "" {
			cfg.Model = value
		}
		if value := strings.TrimSpace(options.VoiceID); value != "" {
			cfg.VoiceID = value
		}
		if value := strings.TrimSpace(options.Language); value != "" {
			cfg.Language = value
		}
	}
	if value := strings.TrimSpace(req.Voice); value != "" {
		cfg.VoiceID = value
	}
	if value := strings.TrimSpace(req.Language); value != "" {
		cfg.Language = value
	}

	client := p.Client
	if client == nil {
		client = NewClient(nil)
	}
	stream, err := client.StartSynthesis(ctx, cfg, req.Format)
	if err != nil {
		return nil, err
	}

	return &providerTTSStream{
		stream: stream,
		events: mapTTSEvents(ctx, stream.Events()),
	}, nil
}

type providerTTSStream struct {
	stream TextStream
	events <-chan providersdk.TTSEvent
}

func (s *providerTTSStream) SendText(
	ctx context.Context,
	text string,
	more bool,
) error {
	return s.stream.SendText(ctx, text, more)
}

func (s *providerTTSStream) Events() <-chan providersdk.TTSEvent {
	return s.events
}

func (s *providerTTSStream) Close() error {
	return s.stream.Close()
}

func mapTTSEvents(
	ctx context.Context,
	source <-chan Event,
) <-chan providersdk.TTSEvent {
	events := make(chan providersdk.TTSEvent, 32)
	go func() {
		defer close(events)

		for {
			select {
			case event, ok := <-source:
				if !ok {
					return
				}

				mapped := providersdk.TTSEvent{
					Audio:      event.Audio,
					ProviderID: event.RequestID,
					Done:       event.Done,
					Err:        event.Err,
				}
				select {
				case events <- mapped:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return events
}

var _ providersdk.TTS = Provider{}
