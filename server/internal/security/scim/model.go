package scim

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	userSchema       = "urn:ietf:params:scim:schemas:core:2.0:User"
	groupSchema      = "urn:ietf:params:scim:schemas:core:2.0:Group"
	listSchema       = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	patchSchema      = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	errorSchema      = "urn:ietf:params:scim:api:messages:2.0:Error"
	defaultPageCount = 100
	maxPageCount     = 200
)

type Token struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Name           string     `json:"name"`
	Prefix         string     `json:"prefix"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreatedToken struct {
	Token  Token  `json:"token_metadata"`
	Secret string `json:"token"`
}

type CreateTokenRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type User struct {
	Schemas     []string `json:"schemas"`
	ID          string   `json:"id"`
	ExternalID  string   `json:"externalId,omitempty"`
	UserName    string   `json:"userName"`
	DisplayName string   `json:"displayName,omitempty"`
	Active      bool     `json:"active"`
	Meta        Meta     `json:"meta"`
}

type UserInput struct {
	Schemas     []string `json:"schemas,omitempty"`
	ExternalID  string   `json:"externalId,omitempty"`
	UserName    string   `json:"userName"`
	DisplayName string   `json:"displayName,omitempty"`
	Active      *bool    `json:"active,omitempty"`
}

type Group struct {
	Schemas     []string      `json:"schemas"`
	ID          string        `json:"id"`
	ExternalID  string        `json:"externalId,omitempty"`
	DisplayName string        `json:"displayName"`
	Members     []GroupMember `json:"members"`
	Meta        Meta          `json:"meta"`
}

type GroupInput struct {
	Schemas     []string      `json:"schemas,omitempty"`
	ExternalID  string        `json:"externalId,omitempty"`
	DisplayName string        `json:"displayName"`
	Members     []GroupMember `json:"members,omitempty"`
}

type GroupMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

type Meta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
}

type UserRecord struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	ExternalID     string
	UserName       string
	DisplayName    *string
	Active         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type GroupRecord struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ExternalID     *string
	DisplayName    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int64    `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    any      `json:"Resources"`
}

type PatchRequest struct {
	Schemas    []string         `json:"schemas"`
	Operations []PatchOperation `json:"Operations"`
}

type PatchOperation struct {
	Op    string          `json:"op"`
	Path  string          `json:"path,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

type ErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}
