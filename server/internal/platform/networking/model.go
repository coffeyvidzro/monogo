package networking

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

const (
	ActionAllow    = "allow"
	ActionDeny     = "deny"
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type Policy struct {
	ID             uuid.UUID    `json:"id"`
	OrganizationID uuid.UUID    `json:"organization_id"`
	Name           string       `json:"name"`
	Action         string       `json:"action"`
	SourceCIDR     netip.Prefix `json:"source_cidr"`
	Status         string       `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type CreateRequest struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	SourceCIDR string `json:"source_cidr"`
}

type UpdateRequest struct {
	Name       *string `json:"name,omitempty"`
	Action     *string `json:"action,omitempty"`
	SourceCIDR *string `json:"source_cidr,omitempty"`
	Status     *string `json:"status,omitempty"`
}
