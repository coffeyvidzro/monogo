package providers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	ProviderOpenAI   = "openai"
	ProviderDeepgram = "deepgram"
	ProviderGroq     = "groq"
	ProviderCartesia = "cartesia"

	RoleRealtime = "realtime"
	RoleSTT      = "stt"
	RoleLLM      = "llm"
	RoleTTS      = "tts"
)

type Credential struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Provider       string
	Name           string
	CreatedAt      time.Time
	RotatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateCredentialRequest struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Secret   string `json:"secret"`
}

type RotateCredentialRequest struct {
	Secret string `json:"secret"`
}

type CredentialResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Provider       string    `json:"provider"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	RotatedAt      time.Time `json:"rotated_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Binding struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	VoiceAgentID   uuid.UUID
	Role           string
	Provider       string
	CredentialID   uuid.UUID
	Config         json.RawMessage
}

type UpsertBindingRequest struct {
	Provider     string          `json:"provider"`
	CredentialID uuid.UUID       `json:"credential_id"`
	Config       json.RawMessage `json:"config,omitempty"`
}

type BindingResponse struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	VoiceAgentID   uuid.UUID       `json:"voice_agent_id"`
	Role           string          `json:"role"`
	Provider       string          `json:"provider"`
	CredentialID   uuid.UUID       `json:"credential_id"`
	Config         json.RawMessage `json:"config"`
}

type ResolvedBinding struct {
	Role     string
	Provider string
	APIKey   string
	Config   json.RawMessage
}

func credentialResponse(value Credential) CredentialResponse {
	return CredentialResponse{
		ID: value.ID,
		OrganizationID: value.OrganizationID,
		Provider: value.Provider,
		Name: value.Name,
		CreatedAt: value.CreatedAt,
		RotatedAt: value.RotatedAt,
		UpdatedAt: value.UpdatedAt,
	}
}

func bindingResponse(value Binding) BindingResponse {
	return BindingResponse{
		ID: value.ID,
		OrganizationID: value.OrganizationID,
		VoiceAgentID: value.VoiceAgentID,
		Role: value.Role,
		Provider: value.Provider,
		CredentialID: value.CredentialID,
		Config: value.Config,
	}
}
