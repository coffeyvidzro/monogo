package networking

import (
	"net/netip"
	"testing"
)

func TestAllowsUsesDenyPrecedenceAndAllowDefault(t *testing.T) {
	address := netip.MustParseAddr("192.0.2.10")
	policies := []Policy{
		{
			Action:     ActionAllow,
			Status:     StatusActive,
			SourceCIDR: netip.MustParsePrefix("192.0.2.0/24"),
		},
		{
			Action:     ActionDeny,
			Status:     StatusActive,
			SourceCIDR: netip.MustParsePrefix("192.0.2.10/32"),
		},
	}
	if Allows(policies, address) {
		t.Fatal("Allows() = true for matching deny")
	}
	if Allows(policies[:1], netip.MustParseAddr("198.51.100.1")) {
		t.Fatal("Allows() = true for unmatched address with active allow list")
	}
	if !Allows(nil, address) {
		t.Fatal("Allows() = false without policies")
	}
}
