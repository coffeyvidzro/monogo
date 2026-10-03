package networking

import (
	"net/netip"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func validateCreate(req CreateRequest) (CreateRequest, netip.Prefix, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Action = strings.TrimSpace(req.Action)
	if req.Name == "" || len(req.Name) > 128 {
		return CreateRequest{}, netip.Prefix{}, apperror.NewBadRequest("name must be between 1 and 128 characters")
	}
	if req.Action != ActionAllow && req.Action != ActionDeny {
		return CreateRequest{}, netip.Prefix{}, apperror.NewBadRequest("action must be allow or deny")
	}
	prefix, err := netip.ParsePrefix(strings.TrimSpace(req.SourceCIDR))
	if err != nil {
		return CreateRequest{}, netip.Prefix{}, apperror.NewBadRequest("source_cidr must be a valid CIDR")
	}
	return req, prefix.Masked(), nil
}

func validateUpdate(req UpdateRequest) (UpdateRequest, *netip.Prefix, error) {
	if req.Name == nil && req.Action == nil && req.SourceCIDR == nil && req.Status == nil {
		return UpdateRequest{}, nil, apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" || len(value) > 128 {
			return UpdateRequest{}, nil, apperror.NewBadRequest("name must be between 1 and 128 characters")
		}
		req.Name = &value
	}
	if req.Action != nil {
		value := strings.TrimSpace(*req.Action)
		if value != ActionAllow && value != ActionDeny {
			return UpdateRequest{}, nil, apperror.NewBadRequest("action must be allow or deny")
		}
		req.Action = &value
	}
	if req.Status != nil {
		value := strings.TrimSpace(*req.Status)
		if value != StatusActive && value != StatusDisabled {
			return UpdateRequest{}, nil, apperror.NewBadRequest("status must be active or disabled")
		}
		req.Status = &value
	}
	var prefix *netip.Prefix
	if req.SourceCIDR != nil {
		value, err := netip.ParsePrefix(strings.TrimSpace(*req.SourceCIDR))
		if err != nil {
			return UpdateRequest{}, nil, apperror.NewBadRequest("source_cidr must be a valid CIDR")
		}
		value = value.Masked()
		prefix = &value
		canonical := value.String()
		req.SourceCIDR = &canonical
	}
	return req, prefix, nil
}
