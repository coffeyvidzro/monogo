package credentials

import (
	"time"

	"github.com/google/uuid"
)

// Credential is an organization-owned API credential. The secret itself is
// never persisted; only its hash and a non-secret prefix are retained.
type Credential struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CreatedBy      *uuid.UUID
	Name           string
	Description    *string
	TokenPrefix    string
	Scopes         []string
	LastUsedAt     *time.Time
	LastUsedIP     string
	ExpiresAt      *time.Time
	DisabledAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateInput contains the caller-controlled fields for creating a credential.
type CreateInput struct {
	OrganizationID uuid.UUID
	CreatedBy      uuid.UUID
	Name           string
	Description    *string
	Scopes         []string
	ExpiresAt      *time.Time
}

// UpdateInput contains fields that may be changed on an existing credential.
type UpdateInput struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           *string
	Description    *string
	Scopes         *[]string
	ExpiresAt      *time.Time
}

// CreatedCredential is returned at creation time because the plaintext token
// is only available once. It must not be persisted or logged.
type CreatedCredential struct {
	Credential Credential
	Token      string
}

type createRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type updateRequest struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Scopes      *[]string  `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at"`
}
