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
)

type CreateRequest struct {
	Name         string  `json:"name"`
	Engine       string  `json:"engine"`
	Instructions string  `json:"instructions"`
	Voice        *string `json:"voice,omitempty"`
	Language     *string         `json:"language,omitempty"`
	EngineConfig json.RawMessage `json:"engine_config,omitempty"`
}

type UpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	Engine       *string `json:"engine,omitempty"`
	Instructions *string `json:"instructions,omitempty"`
	Voice        *string `json:"voice,omitempty"`
	Language     *string          `json:"language,omitempty"`
	EngineConfig *json.RawMessage `json:"engine_config,omitempty"`
}

type Response struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Engine         string    `json:"engine"`
	Instructions   string    `json:"instructions"`
	Voice          *string   `json:"voice,omitempty"`
	Language       *string   `json:"language,omitempty"`
	Status         string          `json:"status"`
	EngineConfig   json.RawMessage `json:"engine_config"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func response(agent sqlc.VoiceAgent) Response {
	return Response{
		ID: agent.ID, OrganizationID: agent.OrganizationID, Name: agent.Name,
		Engine: agent.Engine, Instructions: agent.Instructions, Voice: agent.Voice,
		Language: agent.Language, Status: agent.Status, EngineConfig: json.RawMessage(agent.EngineConfig),
		CreatedAt: pgconv.TimestamptzToTime(agent.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(agent.UpdatedAt),
	}
}
