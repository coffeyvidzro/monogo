package orchestration

import (
	"context"
	"encoding/json"

	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func (s *Service) ExecuteTool(
	ctx context.Context,
	req tools.ExecuteRequest,
) (tools.ExecuteResult, error) {
	return s.tools.Execute(ctx, req)
}

func (s *Service) MediaTools(
	ctx context.Context,
	organizationID uuid.UUID,
	voiceAgentID uuid.UUID,
) ([]session.Tool, error) {
	values, err := s.tools.List(ctx, organizationID, voiceAgentID)
	if err != nil {
		return nil, err
	}
	result := make([]session.Tool, 0, len(values))
	for _, value := range values {
		if !value.Enabled {
			continue
		}
		result = append(result, session.Tool{
			ID:          value.ID,
			Name:        value.Name,
			Description: value.Description,
			Parameters:  append(json.RawMessage(nil), value.Parameters...),
		})
	}
	return result, nil
}

func (s *Service) ExecuteNamedTool(
	ctx context.Context,
	req tools.ExecuteNamedRequest,
) (tools.ExecuteResult, error) {
	return s.tools.ExecuteNamed(ctx, req)
}
