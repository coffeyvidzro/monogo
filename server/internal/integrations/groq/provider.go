package groq

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
		ID:   "groq",
		Kind: providersdk.KindLLM,
		Capabilities: []providersdk.Capability{
			providersdk.CapabilityStreaming,
			providersdk.CapabilityToolCalling,
			providersdk.CapabilityUsage,
		},
	}
}

func (p Provider) Generate(
	ctx context.Context,
	req providersdk.LLMRequest,
) (providersdk.LLMStream, error) {
	cfg := p.Config
	if strings.TrimSpace(req.Runtime.APIKey) != "" {
		cfg.APIKey = req.Runtime.APIKey
	}
	if len(req.Runtime.Config) != 0 {
		var options struct {
			Endpoint            string   `json:"endpoint"`
			Model               string   `json:"model"`
			Temperature         *float64 `json:"temperature"`
			MaxCompletionTokens int      `json:"max_completion_tokens"`
		}
		if err := json.Unmarshal(req.Runtime.Config, &options); err != nil {
			return nil, fmt.Errorf("decode Groq provider config: %w", err)
		}
		if value := strings.TrimSpace(options.Endpoint); value != "" {
			cfg.Endpoint = value
		}
		if value := strings.TrimSpace(options.Model); value != "" {
			cfg.Model = value
		}
		if options.Temperature != nil {
			cfg.Temperature = options.Temperature
		}
		if options.MaxCompletionTokens > 0 {
			cfg.MaxCompletionTokens = options.MaxCompletionTokens
		}
	}

	cfg.Tools = make([]Tool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		cfg.Tools = append(cfg.Tools, Tool{
			Type: "function",
			Function: FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	messages := make([]Message, 0, len(req.Messages))
	for _, message := range req.Messages {
		native := Message{
			Role:       message.Role,
			Content:    message.Content,
			ToolCallID: message.ToolCallID,
		}
		for _, call := range message.ToolCalls {
			native.ToolCalls = append(native.ToolCalls, MessageToolCall{
				ID:   call.ID,
				Type: "function",
				Function: MessageFunctionCall{
					Name:      call.Name,
					Arguments: string(call.Arguments),
				},
			})
		}
		messages = append(messages, native)
	}

	client := p.Client
	if client == nil {
		client = NewClient(nil)
	}
	stream, err := client.Generate(ctx, cfg, messages)
	if err != nil {
		return nil, err
	}

	return &providerLLMStream{
		stream: stream,
		events: mapLLMEvents(ctx, stream.Events()),
	}, nil
}

type providerLLMStream struct {
	stream Stream
	events <-chan providersdk.LLMEvent
}

func (s *providerLLMStream) Events() <-chan providersdk.LLMEvent {
	return s.events
}

func (s *providerLLMStream) Close() error {
	return s.stream.Close()
}

func mapLLMEvents(
	ctx context.Context,
	source <-chan Event,
) <-chan providersdk.LLMEvent {
	events := make(chan providersdk.LLMEvent, 32)
	go func() {
		defer close(events)

		for {
			select {
			case event, ok := <-source:
				if !ok {
					return
				}

				mapped := providersdk.LLMEvent{
					ResponseID:    event.CompletionID,
					TextDelta:     event.TextDelta,
					ToolCallID:    event.ToolCallID,
					ToolIndex:     event.ToolIndex,
					ToolName:      event.ToolName,
					ToolArguments: append([]byte(nil), event.ToolArguments...),
					Done:          event.Done,
					Err:           event.Err,
				}
				if event.Usage != nil {
					mapped.InputTokens = event.Usage.PromptTokens
					mapped.OutputTokens = event.Usage.CompletionTokens
					mapped.TotalTokens = event.Usage.TotalTokens
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

var _ providersdk.LLM = Provider{}
