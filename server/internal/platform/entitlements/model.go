package entitlements

import (
	"time"

	"github.com/google/uuid"
)

// Capability identifies an optional product capability. Capabilities apply to
// organizations on either Self-Hosted or Cloud; they do not select a deployment
// architecture.
type Capability string

const (
	CapabilitySSO               Capability = "sso"
	CapabilitySCIM              Capability = "scim"
	CapabilityAdvancedRBAC      Capability = "advanced_rbac"
	CapabilityRetentionPolicies Capability = "retention_policies"
	CapabilityPrivateNetworking Capability = "private_networking"
)

type Entitlement struct {
	OrganizationID uuid.UUID
	Capability     Capability
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
