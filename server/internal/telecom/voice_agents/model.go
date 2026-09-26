package voice_agents

import (
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
	Language     *string `json:"language,omitempty"`
}

type UpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	Engine       *string `json:"engine,omitempty"`
	Instructions *string `json:"instructions,omitempty"`
	Voice        *string `json:"voice,omitempty"`
	Language     *string `json:"language,omitempty"`
}

type CreateBindingRequest struct {
	VoiceApplicationID uuid.UUID `json:"voice_application_id"`
}

type Response struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Engine         string    `json:"engine"`
	Instructions   string    `json:"instructions"`
	Voice          *string   `json:"voice,omitempty"`
	Language       *string   `json:"language,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BindingResponse struct {
	ID                 uuid.UUID `json:"id"`
	OrganizationID     uuid.UUID `json:"organization_id"`
	VoiceAgentID       uuid.UUID `json:"voice_agent_id"`
	VoiceApplicationID uuid.UUID `json:"voice_application_id"`
	CreatedAt          time.Time `json:"created_at"`
}

func response(agent sqlc.VoiceAgent) Response {
	return Response{
		ID: agent.ID, OrganizationID: agent.OrganizationID, Name: agent.Name,
		Engine: agent.Engine, Instructions: agent.Instructions, Voice: agent.Voice,
		Language: agent.Language, Status: agent.Status,
		CreatedAt: pgconv.TimestamptzToTime(agent.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(agent.UpdatedAt),
	}
}

func bindingResponse(binding sqlc.VoiceAgentBinding) BindingResponse {
	return BindingResponse{
		ID: binding.ID, OrganizationID: binding.OrganizationID,
		VoiceAgentID: binding.VoiceAgentID, VoiceApplicationID: binding.VoiceApplicationID,
		CreatedAt: pgconv.TimestamptzToTime(binding.CreatedAt),
	}
}
