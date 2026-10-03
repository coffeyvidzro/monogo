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
