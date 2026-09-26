package agents

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func normalizeCreate(req CreateRequest) (CreateRequest, error) {
	name, err := normalizeRequired(req.Name, "name", 255)
	if err != nil {
		return CreateRequest{}, err
	}
	engine, err := normalizeEngine(req.Engine)
	if err != nil {
		return CreateRequest{}, err
	}
	instructions, err := normalizeRequired(req.Instructions, "instructions", 20000)
	if err != nil {
		return CreateRequest{}, err
	}
	voice, err := normalizeOptional(req.Voice, "voice", 255)
	if err != nil {
		return CreateRequest{}, err
	}
	language, err := normalizeOptional(req.Language, "language", 64)
	if err != nil {
		return CreateRequest{}, err
	}
	if len(req.EngineConfig) == 0 {
		req.EngineConfig = json.RawMessage(`{}`)
	}
	if err := validateEngineConfig(req.EngineConfig); err != nil {
		return CreateRequest{}, err
	}
	req.Name, req.Engine, req.Instructions = name, engine, instructions
	req.Voice, req.Language = voice, language
	return req, nil
}

func normalizeUpdate(req UpdateRequest) (UpdateRequest, error) {
	if req.Name == nil && req.Engine == nil && req.Instructions == nil && req.Voice == nil && req.Language == nil && req.EngineConfig == nil {
		return UpdateRequest{}, apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil {
		value, err := normalizeRequired(*req.Name, "name", 255)
		if err != nil {
			return UpdateRequest{}, err
		}
		req.Name = &value
	}
	if req.Engine != nil {
		value, err := normalizeEngine(*req.Engine)
		if err != nil {
			return UpdateRequest{}, err
		}
		req.Engine = &value
	}
	if req.Instructions != nil {
		value, err := normalizeRequired(*req.Instructions, "instructions", 20000)
		if err != nil {
			return UpdateRequest{}, err
		}
		req.Instructions = &value
	}
	var err error
	if req.Voice, err = normalizeOptional(req.Voice, "voice", 255); err != nil {
		return UpdateRequest{}, err
	}
	if req.Language, err = normalizeOptional(req.Language, "language", 64); err != nil {
		return UpdateRequest{}, err
	}
	if req.EngineConfig != nil {
		if err := validateEngineConfig(*req.EngineConfig); err != nil {
			return UpdateRequest{}, err
		}
	}
	return req, nil
}

func validateEngineConfig(value json.RawMessage) error {
	if !json.Valid(value) {
		return apperror.NewBadRequest("engine_config must be valid JSON")
	}
	var object map[string]any
	if err := json.Unmarshal(value, &object); err != nil {
		return apperror.NewBadRequest("engine_config must be a JSON object")
	}
	return nil
}

func validateIDs(organizationID, agentID uuid.UUID) error {
	if organizationID == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	if agentID == uuid.Nil {
		return apperror.NewBadRequest("voice agent id is required")
	}
	return nil
}

func normalizeEngine(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value != EngineComposable && value != EngineIntegrated {
		return "", apperror.NewBadRequest("engine must be composable or integrated")
	}
	return value, nil
}

func normalizeRequired(value, field string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > max {
		return "", apperror.NewBadRequest(field + " must be between 1 and " + strconv.Itoa(max) + " characters")
	}
	return value, nil
}

func normalizeOptional(value *string, field string, max int) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized, err := normalizeRequired(*value, field, max)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}
