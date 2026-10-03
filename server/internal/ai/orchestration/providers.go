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
			Role:     value.Role,
			Provider: value.Provider,
			APIKey:   value.APIKey,
			Config:   append([]byte(nil), value.Config...),
		})
	}
	return out, nil
}

func (s *Service) ProviderRuntimesFromSnapshot(
	ctx context.Context,
	organizationID uuid.UUID,
	snapshot []byte,
) ([]session.ProviderRuntime, error) {
	if s.providers == nil {
		return nil, nil
	}
	values, err := s.providers.ResolveSnapshot(ctx, organizationID, snapshot)
	if err != nil {
		return nil, err
	}
	out := make([]session.ProviderRuntime, 0, len(values))
	for _, value := range values {
		out = append(out, session.ProviderRuntime{
			Role:     value.Role,
			Provider: value.Provider,
			APIKey:   value.APIKey,
			Config:   append([]byte(nil), value.Config...),
		})
	}
	return out, nil
}

func (s *Service) ValidateProviderTopology(
	ctx context.Context,
	organizationID, voiceAgentID uuid.UUID,
	engine session.Engine,
) error {
	if s.providers == nil {
		return nil
	}
	values, err := s.ProviderRuntimes(ctx, organizationID, voiceAgentID)
	if err != nil {
		return err
	}
	return validateProviderTopology(values, engine)
}

func validateProviderTopology(values []session.ProviderRuntime, engine session.Engine) error {
	roles := make(map[string]string, len(values))
	for _, value := range values {
		roles[value.Role] = value.Provider
	}
	switch engine {
	case session.EngineIntegrated:
		if roles["realtime"] != "openai" {
			return apperror.NewBadRequest(
				"Voice Agent requires an OpenAI realtime provider binding",
			)
		}
	case session.EngineComposable:
		if roles["stt"] != "deepgram" {
			return apperror.NewBadRequest(
				"Voice Agent requires a Deepgram STT provider binding",
			)
		}
		if roles["llm"] != "groq" {
			return apperror.NewBadRequest(
				"Voice Agent requires a Groq LLM provider binding",
			)
		}
		if roles["tts"] != "cartesia" {
			return apperror.NewBadRequest(
				"Voice Agent requires a Cartesia TTS provider binding",
			)
		}
	}
	return nil
}

func ValidateProviderRuntimeTopology(
	values []session.ProviderRuntime,
	engine session.Engine,
) error {
	return validateProviderTopology(values, engine)
}
