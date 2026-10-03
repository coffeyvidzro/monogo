package entitlements

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func (c Capability) IsValid() bool {
	switch c {
	case CapabilitySSO,
		CapabilitySCIM,
		CapabilityAdvancedRBAC,
		CapabilityRetentionPolicies,
		CapabilityPrivateNetworking:
		return true
	default:
		return false
	}
}

func normalizeCapability(value Capability) (Capability, error) {
	capability := Capability(strings.TrimSpace(string(value)))
	if !capability.IsValid() {
		return "", apperror.NewBadRequest("unsupported organization capability")
	}
	return capability, nil
}
