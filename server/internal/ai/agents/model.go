// Package agents owns durable AI agent configuration.
package agents

import (
	"encoding/json"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

const (
	EngineComposable = "composable"
	EngineIntegrated = "integrated"

	InterruptionAllow    = "allow"
	InterruptionDisabled = "disabled"
	RecordingNone        = "none"
	RecordingAll         = "all"
)

type ProviderBindingRequest struct {
	Role          string          `json:"role"`
	Provider      string          `json:"provider"`
	IntegrationID uuid.UUID       `json:"integration_id"`
	Config        json.RawMessage `json:"config,omitempty"`
}

type ReadinessIssue struct {
	Code        string `json:"code"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

type ReadinessReport struct {
	Ready                 bool                `json:"ready"`
	ConfigurationRevision int32               `json:"configuration_revision"`
	ActiveRevision        *int32              `json:"active_revision,omitempty"`
	Engine                string              `json:"engine"`
	Preset                *string             `json:"preset,omitempty"`
	PresetVersion         *int32              `json:"preset_version,omitempty"`
	Bindings              []BindingDiagnostic `json:"bindings"`
	Issues                []ReadinessIssue    `json:"issues"`
}

type BindingDiagnostic struct {
	Role            string          `json:"role"`
	Provider        string          `json:"provider"`
	IntegrationID   uuid.UUID       `json:"integration_id"`
	ConnectionState string          `json:"connection_state"`
	FailureCode     *string         `json:"failure_code,omitempty"`
	Config          json.RawMessage `json:"config"`
}

type CreateRequest struct {
	Name               string                   `json:"name"`
	Engine             string                   `json:"engine"`
	Instructions       string                   `json:"instructions"`
	Voice              *string                  `json:"voice,omitempty"`
	Language           *string                  `json:"language,omitempty"`
	EngineConfig       json.RawMessage          `json:"engine_config,omitempty"`
	Preset             *string                  `json:"preset,omitempty"`
	Bindings           []ProviderBindingRequest `json:"bindings,omitempty"`
	InterruptionPolicy string                   `json:"interruption_policy,omitempty"`
	RecordingPolicy    string                   `json:"recording_policy,omitempty"`
	presetVersion      *int32
}

type UpdateRequest struct {
	Name               *string                   `json:"name,omitempty"`
	Engine             *string                   `json:"engine,omitempty"`
	Instructions       *string                   `json:"instructions,omitempty"`
	Voice              *string                   `json:"voice,omitempty"`
	Language           *string                   `json:"language,omitempty"`
	EngineConfig       *json.RawMessage          `json:"engine_config,omitempty"`
	Preset             *string                   `json:"preset,omitempty"`
	Bindings           *[]ProviderBindingRequest `json:"bindings,omitempty"`
	InterruptionPolicy *string                   `json:"interruption_policy,omitempty"`
	RecordingPolicy    *string                   `json:"recording_policy,omitempty"`
	presetVersion      *int32
	updatePreset       bool
}

type Response struct {
	ID                    uuid.UUID       `json:"id"`
	OrganizationID        uuid.UUID       `json:"organization_id"`
	Name                  string          `json:"name"`
	Engine                string          `json:"engine"`
	Instructions          string          `json:"instructions"`
	Voice                 *string         `json:"voice,omitempty"`
	Language              *string         `json:"language,omitempty"`
	Status                string          `json:"status"`
	EngineConfig          json.RawMessage `json:"engine_config"`
	Preset                *string         `json:"preset,omitempty"`
	PresetVersion         *int32          `json:"preset_version,omitempty"`
	InterruptionPolicy    string          `json:"interruption_policy"`
	RecordingPolicy       string          `json:"recording_policy"`
	ConfigurationRevision int32           `json:"configuration_revision"`
	ActiveRevision        *int32          `json:"active_revision,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

func response(agent sqlc.VoiceAgent) Response {
	return Response{
		ID:                    agent.ID,
		OrganizationID:        agent.OrganizationID,
		Name:                  agent.Name,
		Engine:                agent.Engine,
		Instructions:          agent.Instructions,
		Voice:                 agent.Voice,
		Language:              agent.Language,
		Status:                agent.Status,
		EngineConfig:          json.RawMessage(agent.EngineConfig),
		Preset:                agent.Preset,
		PresetVersion:         agent.PresetVersion,
		InterruptionPolicy:    agent.InterruptionPolicy,
		RecordingPolicy:       agent.RecordingPolicy,
		ConfigurationRevision: agent.ConfigurationRevision,
		ActiveRevision:        agent.ActiveRevision,
		CreatedAt:             pgconv.TimestamptzToTime(agent.CreatedAt),
		UpdatedAt:             pgconv.TimestamptzToTime(agent.UpdatedAt),
	}
}
