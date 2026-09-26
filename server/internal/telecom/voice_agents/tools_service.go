package voice_agents

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func (s *Service) CreateTool(ctx context.Context, organizationID, agentID uuid.UUID, req CreateToolRequest) (sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	normalized, err := normalizeCreateTool(req)
	if err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	tool, err := s.repo.CreateTool(ctx, organizationID, agentID, normalized)
	if conflict(err) {
		return sqlc.VoiceAgentTool{}, apperror.NewConflict("voice agent tool already exists")
	}
	if err != nil {
		return sqlc.VoiceAgentTool{}, readError(err, "create voice agent tool")
	}
	return tool, nil
}

func (s *Service) ListTools(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return nil, err
	}
	return s.repo.ListTools(ctx, organizationID, agentID)
}

func (s *Service) UpdateTool(ctx context.Context, organizationID, agentID, id uuid.UUID, req UpdateToolRequest) (sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	if id == uuid.Nil {
		return sqlc.VoiceAgentTool{}, apperror.NewBadRequest("tool id is required")
	}
	normalized, err := normalizeUpdateTool(req)
	if err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	tool, err := s.repo.UpdateTool(ctx, organizationID, agentID, id, normalized)
	if conflict(err) {
		return sqlc.VoiceAgentTool{}, apperror.NewConflict("voice agent tool already exists")
	}
	return tool, readError(err, "voice agent tool not found")
}

func (s *Service) DeleteTool(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	if err := validateIDs(organizationID, agentID); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("tool id is required")
	}
	return writeError(s.repo.DeleteTool(ctx, organizationID, agentID, id), "delete voice agent tool")
}

func normalizeCreateTool(req CreateToolRequest) (CreateToolRequest, error) {
	if req.Type == "" {
		req.Type = ToolTypeWebhook
	}
	req.Type = strings.TrimSpace(req.Type)
	if req.Type != ToolTypeBuiltin && req.Type != ToolTypeWebhook {
		return CreateToolRequest{}, apperror.NewBadRequest("tool type must be builtin or webhook")
	}
	var err error
	req.Name, err = normalizeRequired(req.Name, "name", 128)
	if err != nil {
		return CreateToolRequest{}, err
	}
	req.Description, err = normalizeRequired(req.Description, "description", 2000)
	if err != nil {
		return CreateToolRequest{}, err
	}
	if len(req.Parameters) == 0 {
		req.Parameters = json.RawMessage(`{}`)
	}
	if !json.Valid(req.Parameters) {
		return CreateToolRequest{}, apperror.NewBadRequest("parameters must be valid JSON")
	}
	var parameters map[string]any
	if err := json.Unmarshal(req.Parameters, &parameters); err != nil {
		return CreateToolRequest{}, apperror.NewBadRequest("parameters must be a JSON object")
	}
	if req.TimeoutMS != nil && (*req.TimeoutMS < 100 || *req.TimeoutMS > 30000) {
		return CreateToolRequest{}, apperror.NewBadRequest("timeout_ms must be between 100 and 30000")
	}
	if err := validateToolEndpoint(req.Type, req.EndpointURL); err != nil {
		return CreateToolRequest{}, err
	}
	return req, nil
}

func normalizeUpdateTool(req UpdateToolRequest) (UpdateToolRequest, error) {
	if req.Name == nil && req.Description == nil && req.Parameters == nil && req.EndpointURL == nil && req.TimeoutMS == nil && req.Enabled == nil {
		return UpdateToolRequest{}, apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil {
		value, err := normalizeRequired(*req.Name, "name", 128)
		if err != nil {
			return UpdateToolRequest{}, err
		}
		req.Name = &value
	}
	if req.Description != nil {
		value, err := normalizeRequired(*req.Description, "description", 2000)
		if err != nil {
			return UpdateToolRequest{}, err
		}
		req.Description = &value
	}
	if req.Parameters != nil {
		if !json.Valid(*req.Parameters) {
			return UpdateToolRequest{}, apperror.NewBadRequest("parameters must be valid JSON")
		}
		var parameters map[string]any
		if err := json.Unmarshal(*req.Parameters, &parameters); err != nil {
			return UpdateToolRequest{}, apperror.NewBadRequest("parameters must be a JSON object")
		}
	}
	if req.TimeoutMS != nil && (*req.TimeoutMS < 100 || *req.TimeoutMS > 30000) {
		return UpdateToolRequest{}, apperror.NewBadRequest("timeout_ms must be between 100 and 30000")
	}
	return req, nil
}

func validateToolEndpoint(toolType string, endpoint *string) error {
	if toolType == ToolTypeBuiltin {
		if endpoint != nil {
			return apperror.NewBadRequest("builtin tools cannot define endpoint_url")
		}
		return nil
	}
	if endpoint == nil {
		return apperror.NewBadRequest("webhook tools require endpoint_url")
	}
	value := strings.TrimSpace(*endpoint)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return apperror.NewBadRequest("endpoint_url must be a valid HTTP or HTTPS URL")
	}
	*endpoint = value
	return nil
}
