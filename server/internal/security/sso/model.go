package sso

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	ProtocolSAML = "saml"
	ProtocolOIDC = "oidc"

	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type Connection struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	Name           string          `json:"name"`
	Protocol       string          `json:"protocol"`
	Issuer         string          `json:"issuer"`
	Configuration  json.RawMessage `json:"configuration"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Name          string          `json:"name"`
	Protocol      string          `json:"protocol"`
	Issuer        string          `json:"issuer"`
	Configuration json.RawMessage `json:"configuration,omitempty"`
	Secret        *string         `json:"secret,omitempty"`
}

type UpdateRequest struct {
	Name          *string          `json:"name,omitempty"`
	Issuer        *string          `json:"issuer,omitempty"`
	Configuration *json.RawMessage `json:"configuration,omitempty"`
	Secret        *string          `json:"secret,omitempty"`
	Status        *string          `json:"status,omitempty"`
}
