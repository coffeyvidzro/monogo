package orchestration

import (
	"context"
	"encoding/json"

	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func (s *Service) ExecuteTool(
	ctx context.Context,
	req tools.ExecuteRequest,
) (tools.ExecuteResult, error) {
	return s.tools.Execute(ctx, req)
}

func (s *Service) ToolDefinitions(
	ctx context.Context,
	organizationID, voiceAgentID uuid.UUID,
) ([]session.ToolDefinition, error) {
	items, err := s.tools.List(ctx, organizationID, voiceAgentID)
	if err != nil {
		return nil, err
	}
	result := make([]session.ToolDefinition, 0, len(items))
	for _, tool := range items {
		if !tool.Enabled {
			continue
		}
		result = append(result, session.ToolDefinition{
			ID:          tool.ID,
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  json.RawMessage(append([]byte(nil), tool.Parameters...)),
		})
	}
	return result, nil
}

func (s *Service) ResolveToolByName(
	ctx context.Context,
	organizationID, voiceAgentID uuid.UUID,
	name string,
) (sqlc.VoiceAgentTool, error) {
	return s.tools.ResolveByName(ctx, organizationID, voiceAgentID, name)
}
