package orchestration

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func (s *Service) ProviderRuntimes(
	ctx context.Context,
	organizationID, voiceAgentID uuid.UUID,
) ([]session.ProviderRuntime, error) {
	if s.providers == nil {
		return nil, nil
	}
	values, err := s.providers.Resolve(ctx, organizationID, voiceAgentID)
	if err != nil {
		return nil, err
	}
	out := make([]session.ProviderRuntime, 0, len(values))
	for _, value := range values {
		out = append(out, session.ProviderRuntime{
			Role: value.Role,
			Provider: value.Provider,
			APIKey: value.APIKey,
			Config: append([]byte(nil), value.Config...),
		})
	}
	return out, nil
}

func (s *Service) ValidateProviderTopology(
	ctx context.Context,
	organizationID, voiceAgentID uuid.UUID,
	engine session.Engine,
) error {
	values, err := s.ProviderRuntimes(ctx, organizationID, voiceAgentID)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	roles := make(map[string]string, len(values))
	for _, value := range values {
		roles[value.Role] = value.Provider
	}
	switch engine {
	case session.EngineIntegrated:
		if roles["realtime"] != "openai" {
			return apperror.NewBadRequest("integrated engine requires an OpenAI realtime provider binding")
		}
	case session.EngineComposable:
		if roles["stt"] != "deepgram" || roles["llm"] != "groq" || roles["tts"] != "cartesia" {
			return apperror.NewBadRequest("composable engine requires Deepgram STT, Groq LLM, and Cartesia TTS bindings")
		}
	}
	return nil
}
