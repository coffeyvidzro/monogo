package tools

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Create(ctx context.Context, organizationID, agentID uuid.UUID, req CreateRequest) (sqlc.VoiceAgentTool, error) {
	timeout := int32(3000)
	if req.TimeoutMS != nil { timeout = *req.TimeoutMS }
	enabled := true
	if req.Enabled != nil { enabled = *req.Enabled }
	return r.queries.CreateVoiceAgentTool(ctx, sqlc.CreateVoiceAgentToolParams{
		OrganizationID: organizationID, VoiceAgentID: agentID, Type: req.Type,
		Name: req.Name, Description: req.Description, Parameters: []byte(req.Parameters),
		EndpointUrl: req.EndpointURL, TimeoutMs: timeout, Enabled: enabled,
	})
}

func (r *Repository) List(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentTool, error) {
	return r.queries.ListVoiceAgentToolsByAgentID(ctx, sqlc.ListVoiceAgentToolsByAgentIDParams{
		OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}

func (r *Repository) Update(ctx context.Context, organizationID, agentID, id uuid.UUID, req UpdateRequest) (sqlc.VoiceAgentTool, error) {
	var parameters []byte
	if req.Parameters != nil { parameters = []byte(*req.Parameters) }
	return r.queries.UpdateVoiceAgentTool(ctx, sqlc.UpdateVoiceAgentToolParams{
		Name: req.Name, Description: req.Description, Parameters: parameters,
		EndpointUrl: req.EndpointURL, TimeoutMs: req.TimeoutMS, Enabled: req.Enabled,
		ID: id, OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}

func (r *Repository) Delete(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	return r.queries.DeleteVoiceAgentTool(ctx, sqlc.DeleteVoiceAgentToolParams{
		ID: id, OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}
