package deepgram

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
		ID:   "deepgram",
		Kind: providersdk.KindSTT,
		Capabilities: []providersdk.Capability{
			providersdk.CapabilityStreaming,
			providersdk.CapabilityTurnDetection,
		},
	}
}

func (p Provider) StartSTT(
	ctx context.Context,
	runtime providersdk.Runtime,
	format session.AudioFormat,
	language string,
) (providersdk.STTStream, error) {
	cfg := p.Config
	if strings.TrimSpace(runtime.APIKey) != "" {
		cfg.APIKey = runtime.APIKey
	}
	if len(runtime.Config) != 0 {
		var options struct {
			Endpoint string `json:"endpoint"`
			Model    string `json:"model"`
		}
		if err := json.Unmarshal(runtime.Config, &options); err != nil {
			return nil, fmt.Errorf("decode Deepgram provider config: %w", err)
		}
		if value := strings.TrimSpace(options.Endpoint); value != "" {
			cfg.Endpoint = value
		}
		if value := strings.TrimSpace(options.Model); value != "" {
			cfg.Model = value
		}
	}
	if value := strings.TrimSpace(language); value != "" {
		cfg.LanguageHints = append(append([]string(nil), cfg.LanguageHints...), value)
	}

	client := p.Client
	if client == nil {
		client = NewClient(nil)
	}
	stream, err := client.Start(ctx, cfg, format)
	if err != nil {
		return nil, err
	}
	return &providerStream{
		stream: stream,
		events: mapSTTEvents(ctx, stream.Events()),
	}, nil
}

type providerStream struct {
	stream Stream
	events <-chan providersdk.STTEvent
}

func (s *providerStream) SendAudio(
	ctx context.Context,
	frame session.AudioFrame,
) error {
	return s.stream.SendAudio(ctx, frame)
}

func (s *providerStream) Finalize(ctx context.Context) error {
	return s.stream.Finalize(ctx)
}

func (s *providerStream) Events() <-chan providersdk.STTEvent {
	return s.events
}

func (s *providerStream) Close(ctx context.Context) error {
	return s.stream.Close(ctx)
}

func mapSTTEvents(
	ctx context.Context,
	source <-chan Event,
) <-chan providersdk.STTEvent {
	events := make(chan providersdk.STTEvent, 32)
	go func() {
		defer close(events)

		for {
			select {
			case event, ok := <-source:
				if !ok {
					return
				}

				mapped := providersdk.STTEvent{
					ProviderID: event.RequestID,
					Err:        event.Err,
				}
				switch event.TurnEvent {
				case "StartOfTurn":
					mapped.Type = providersdk.STTEventSpeechStarted
				case "EndOfTurn":
					mapped.Type = providersdk.STTEventSpeechStopped
					mapped.Text = event.Transcript.Text
				default:
					if event.Transcript.Text == "" {
						continue
					}
					mapped.Text = event.Transcript.Text
					if event.Transcript.IsFinal || event.Transcript.SpeechFinal {
						mapped.Type = providersdk.STTEventTranscriptFinal
					} else {
						mapped.Type = providersdk.STTEventTranscriptDelta
					}
				}
				if event.Err != nil {
					mapped.Type = providersdk.STTEventError
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

var _ providersdk.STT = Provider{}
