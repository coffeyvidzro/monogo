package runtimes

import (
	"regexp"
	"sort"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

var capabilityPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

func normalizeCreate(req *CreateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 1 || len(req.Name) > 128 {
		return apperror.NewBadRequest("name must be between 1 and 128 characters")
	}
	if req.Region != nil {
		value := strings.TrimSpace(*req.Region)
		if len(value) < 1 || len(value) > 128 {
			return apperror.NewBadRequest("region must be between 1 and 128 characters")
		}
		req.Region = &value
	}
	return nil
}

func normalizeHeartbeat(req *HeartbeatRequest) error {
	req.Version = strings.TrimSpace(req.Version)
	if len(req.Version) < 1 || len(req.Version) > 128 {
		return apperror.NewBadRequest("version must be between 1 and 128 characters")
	}
	if req.Capacity < 0 {
		return apperror.NewBadRequest("capacity cannot be negative")
	}
	if req.ActiveSessions < 0 || req.ActiveSessions > req.Capacity {
		return apperror.NewBadRequest("active_sessions must be between 0 and capacity")
	}
	if len(req.Capabilities) > 32 {
		return apperror.NewBadRequest("capabilities cannot contain more than 32 values")
	}

	seen := make(map[string]struct{}, len(req.Capabilities))
	capabilities := make([]string, 0, len(req.Capabilities))
	for _, capability := range req.Capabilities {
		capability = strings.ToLower(strings.TrimSpace(capability))
		if !capabilityPattern.MatchString(capability) {
			return apperror.NewBadRequest("capabilities contain an invalid value")
		}
		if _, ok := seen[capability]; ok {
			continue
		}
		seen[capability] = struct{}{}
		capabilities = append(capabilities, capability)
	}
	sort.Strings(capabilities)
	req.Capabilities = capabilities
	return nil
}
