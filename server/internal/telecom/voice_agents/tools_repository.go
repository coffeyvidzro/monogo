package voice_agents

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

func (r *Repository) CreateTool(ctx context.Context, organizationID, agentID uuid.UUID, req CreateToolRequest) (sqlc.VoiceAgentTool, error) {
	timeout := int32(3000)
	if req.TimeoutMS != nil {
		timeout = *req.TimeoutMS
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return r.queries.CreateVoiceAgentTool(ctx, sqlc.CreateVoiceAgentToolParams{
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
		Type:           req.Type,
		Name:           req.Name,
		Description:    req.Description,
		Parameters:     []byte(req.Parameters),
		EndpointUrl:    req.EndpointURL,
		TimeoutMs:      timeout,
		Enabled:        enabled,
	})
}

func (r *Repository) ListTools(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentTool, error) {
	return r.queries.ListVoiceAgentToolsByAgentID(ctx, sqlc.ListVoiceAgentToolsByAgentIDParams{
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) GetTool(ctx context.Context, organizationID, agentID, id uuid.UUID) (sqlc.VoiceAgentTool, error) {
	return r.queries.GetVoiceAgentToolByID(ctx, sqlc.GetVoiceAgentToolByIDParams{
		ID:             id,
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) UpdateTool(ctx context.Context, organizationID, agentID, id uuid.UUID, req UpdateToolRequest) (sqlc.VoiceAgentTool, error) {
	var parameters []byte
	if req.Parameters != nil {
		parameters = []byte(*req.Parameters)
	}
	return r.queries.UpdateVoiceAgentTool(ctx, sqlc.UpdateVoiceAgentToolParams{
		Name:           req.Name,
		Description:    req.Description,
		Parameters:     parameters,
		EndpointUrl:    req.EndpointURL,
		TimeoutMs:      req.TimeoutMS,
		Enabled:        req.Enabled,
		ID:             id,
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) DeleteTool(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	return r.queries.DeleteVoiceAgentTool(ctx, sqlc.DeleteVoiceAgentToolParams{
		ID:             id,
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}
