// Package tools owns AI agent tool definitions and execution.
package tools

import (
	"encoding/json"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

const (
	TypeBuiltin = "builtin"
	TypeWebhook = "webhook"
)

type CreateRequest struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	EndpointURL *string         `json:"endpoint_url,omitempty"`
	TimeoutMS   *int32          `json:"timeout_ms,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
}

type UpdateRequest struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Parameters  *json.RawMessage `json:"parameters,omitempty"`
	EndpointURL *string          `json:"endpoint_url,omitempty"`
	TimeoutMS   *int32           `json:"timeout_ms,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"`
}

type Response struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	VoiceAgentID   uuid.UUID       `json:"voice_agent_id"`
	Type           string          `json:"type"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Parameters     json.RawMessage `json:"parameters"`
	EndpointURL    *string         `json:"endpoint_url,omitempty"`
	TimeoutMS      int32           `json:"timeout_ms"`
	Enabled        bool            `json:"enabled"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func response(tool sqlc.VoiceAgentTool) Response {
	return Response{
		ID: tool.ID, OrganizationID: tool.OrganizationID, VoiceAgentID: tool.VoiceAgentID,
		Type: tool.Type, Name: tool.Name, Description: tool.Description,
		Parameters: json.RawMessage(tool.Parameters), EndpointURL: tool.EndpointUrl,
		TimeoutMS: tool.TimeoutMs, Enabled: tool.Enabled,
		CreatedAt: pgconv.TimestamptzToTime(tool.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(tool.UpdatedAt),
	}
}
