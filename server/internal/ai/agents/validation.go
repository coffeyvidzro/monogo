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
	if req.Preset != nil {
		presetID := strings.TrimSpace(*req.Preset)
		preset, ok := presetByID(presetID)
		if !ok {
			return CreateRequest{}, apperror.NewBadRequest("unknown Voice Agent preset")
		}
		req.Preset = &presetID
		req.presetVersion = &preset.Version
		req.Engine = preset.Engine
		req.EngineConfig = append(json.RawMessage(nil), preset.EngineConfig...)
		req.Bindings = applyPresetBindingConfig(req.Bindings, preset)
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
	if req.InterruptionPolicy == "" {
		req.InterruptionPolicy = InterruptionAllow
	}
	if req.RecordingPolicy == "" {
		req.RecordingPolicy = RecordingNone
	}
	if err := validatePolicies(req.InterruptionPolicy, req.RecordingPolicy); err != nil {
		return CreateRequest{}, err
	}
	return req, nil
}

func normalizeUpdate(req UpdateRequest) (UpdateRequest, error) {
	if req.Name == nil && req.Engine == nil && req.Instructions == nil && req.Voice == nil && req.Language == nil && req.EngineConfig == nil && req.Preset == nil && req.Bindings == nil && req.InterruptionPolicy == nil && req.RecordingPolicy == nil {
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
	if req.Preset != nil {
		presetID := strings.TrimSpace(*req.Preset)
		preset, ok := presetByID(presetID)
		if !ok {
			return UpdateRequest{}, apperror.NewBadRequest("unknown Voice Agent preset")
		}
		req.Preset = &presetID
		req.presetVersion = &preset.Version
		req.updatePreset = true
		req.Engine = &preset.Engine
		config := append(json.RawMessage(nil), preset.EngineConfig...)
		req.EngineConfig = &config
		bindings := []ProviderBindingRequest{}
		if req.Bindings != nil {
			bindings = *req.Bindings
		}
		bindings = applyPresetBindingConfig(bindings, preset)
		req.Bindings = &bindings
	} else if req.Engine != nil || req.EngineConfig != nil {
		req.updatePreset = true
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
	if req.InterruptionPolicy != nil || req.RecordingPolicy != nil {
		interruption := InterruptionAllow
		recording := RecordingNone
		if req.InterruptionPolicy != nil {
			interruption = strings.TrimSpace(*req.InterruptionPolicy)
			req.InterruptionPolicy = &interruption
		}
		if req.RecordingPolicy != nil {
			recording = strings.TrimSpace(*req.RecordingPolicy)
			req.RecordingPolicy = &recording
		}
		if err := validatePolicies(interruption, recording); err != nil {
			return UpdateRequest{}, err
		}
	}
	return req, nil
}

func applyPresetBindingConfig(
	bindings []ProviderBindingRequest,
	preset Preset,
) []ProviderBindingRequest {
	result := make([]ProviderBindingRequest, 0, len(bindings))
	for _, binding := range bindings {
		if config, ok := preset.ProviderConfig[binding.Role]; ok && len(binding.Config) == 0 {
			binding.Config = append(json.RawMessage(nil), config...)
		}
		result = append(result, binding)
	}
	return result
}

func validatePolicies(interruption string, recording string) error {
	if interruption != InterruptionAllow && interruption != InterruptionDisabled {
		return apperror.NewBadRequest("interruption_policy must be allow or disabled")
	}
	if recording != RecordingNone && recording != RecordingAll {
		return apperror.NewBadRequest("recording_policy must be none or all")
	}
	return nil
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
