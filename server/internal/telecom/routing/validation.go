package routing

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/uuid"
)

func validateOutboundRequest(req OutboundRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization_id is required")
	}
	if req.TrunkID != nil && *req.TrunkID == uuid.Nil {
		return fmt.Errorf("trunk_id is invalid")
	}
	if strings.TrimSpace(req.From) == "" {
		return fmt.Errorf("from is required")
	}
	if strings.TrimSpace(req.To) == "" {
		return fmt.Errorf("to is required")
	}
	return nil
}

func validateInboundRequest(req InboundRequest) (netip.Addr, error) {
	sourceIP, err := netip.ParseAddr(strings.TrimSpace(req.SourceIP))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid source_ip: %w", err)
	}
	if strings.TrimSpace(req.CalledNumber) == "" {
		return netip.Addr{}, fmt.Errorf("called_number is required")
	}
	if strings.TrimSpace(req.CallerNumber) == "" {
		return netip.Addr{}, fmt.Errorf("caller_number is required")
	}
	return sourceIP, nil
}
