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
	if req.TimeoutMS != nil {
		timeout = *req.TimeoutMS
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return r.queries.CreateVoiceAgentTool(ctx, sqlc.CreateVoiceAgentToolParams{
		OrganizationID: organizationID, VoiceAgentID: agentID, Type: req.Type,
		Name: req.Name, Description: req.Description, Parameters: []byte(req.Parameters),
		EndpointUrl: req.EndpointURL, TimeoutMs: timeout, Enabled: enabled,
	})
}

func (r *Repository) CreateWebhook(
	ctx context.Context,
	organizationID, agentID, id uuid.UUID,
	req CreateRequest,
	secretCiphertext string,
) (sqlc.VoiceAgentTool, error) {
	timeout := int32(3000)
	if req.TimeoutMS != nil {
		timeout = *req.TimeoutMS
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if _, err := r.queries.CreateVoiceAgentWebhookTool(ctx, sqlc.CreateVoiceAgentWebhookToolParams{
		ID:               id,
		OrganizationID:   organizationID,
		VoiceAgentID:     agentID,
		Name:             req.Name,
		Description:      req.Description,
		Parameters:       []byte(req.Parameters),
		EndpointUrl:      *req.EndpointURL,
		TimeoutMs:        timeout,
		Enabled:          enabled,
		SecretCiphertext: secretCiphertext,
	}); err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	return r.Get(ctx, organizationID, agentID, id)
}

func (r *Repository) GetSigningSecret(
	ctx context.Context,
	organizationID, agentID, toolID uuid.UUID,
) (string, error) {
	return r.queries.GetVoiceAgentToolSigningSecret(ctx, sqlc.GetVoiceAgentToolSigningSecretParams{
		ToolID:         toolID,
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) RotateSigningSecret(
	ctx context.Context,
	organizationID, agentID, toolID uuid.UUID,
	secretCiphertext string,
) (int64, error) {
	return r.queries.RotateVoiceAgentToolSigningSecret(ctx, sqlc.RotateVoiceAgentToolSigningSecretParams{
		SecretCiphertext: secretCiphertext,
		ToolID:           toolID,
		OrganizationID:   organizationID,
		VoiceAgentID:     agentID,
	})
}

func (r *Repository) ClaimExecution(
	ctx context.Context,
	req ExecuteRequest,
) (uuid.UUID, error) {
	return r.queries.ClaimVoiceAgentToolExecution(ctx, sqlc.ClaimVoiceAgentToolExecutionParams{
		OrganizationID: req.OrganizationID,
		SessionID:      req.SessionID,
		VoiceAgentID:   req.VoiceAgentID,
		ToolID:         req.ToolID,
		CallID:         req.CallID,
		ToolCallID:     req.ToolCallID,
		Arguments:      []byte(req.Arguments),
	})
}

func (r *Repository) GetExecution(
	ctx context.Context,
	organizationID, sessionID uuid.UUID,
	toolCallID string,
) (sqlc.VoiceAgentToolExecution, error) {
	return r.queries.GetVoiceAgentToolExecution(ctx, sqlc.GetVoiceAgentToolExecutionParams{
		OrganizationID: organizationID,
		SessionID:      sessionID,
		ToolCallID:     toolCallID,
	})
}

func (r *Repository) MarkExecutionSucceeded(
	ctx context.Context,
	organizationID, id uuid.UUID,
	result ExecuteResult,
) (int64, error) {
	status := int32(result.StatusCode)
	contentType := result.ContentType
	return r.queries.MarkVoiceAgentToolExecutionSucceeded(ctx, sqlc.MarkVoiceAgentToolExecutionSucceededParams{
		ResponseStatus:      &status,
		ResponseContentType: &contentType,
		ResponseBody:        result.Body,
		ID:                  id,
		OrganizationID:      organizationID,
	})
}

func (r *Repository) MarkExecutionFailed(
	ctx context.Context,
	organizationID, id uuid.UUID,
	message string,
) (int64, error) {
	return r.queries.MarkVoiceAgentToolExecutionFailed(ctx, sqlc.MarkVoiceAgentToolExecutionFailedParams{
		ErrorMessage:   message,
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Get(ctx context.Context, organizationID, agentID, id uuid.UUID) (sqlc.VoiceAgentTool, error) {
	return r.queries.GetVoiceAgentToolByID(ctx, sqlc.GetVoiceAgentToolByIDParams{
		ID: id, OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}

func (r *Repository) List(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentTool, error) {
	return r.queries.ListVoiceAgentToolsByAgentID(ctx, sqlc.ListVoiceAgentToolsByAgentIDParams{
		OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}

func (r *Repository) Update(ctx context.Context, organizationID, agentID, id uuid.UUID, req UpdateRequest) (sqlc.VoiceAgentTool, error) {
	var parameters []byte
	if req.Parameters != nil {
		parameters = []byte(*req.Parameters)
	}
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
