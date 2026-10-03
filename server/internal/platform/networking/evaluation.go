package networking

import "net/netip"

// Allows applies deterministic policy precedence: a matching deny always wins;
// otherwise a matching allow permits access. When any active allow rule exists,
// unmatched addresses are denied. With deny-only or no active rules, unmatched
// addresses are allowed.
func Allows(policies []Policy, address netip.Addr) bool {
	hasAllow := false
	matchedAllow := false
	for _, policy := range policies {
		if policy.Status != StatusActive {
			continue
		}
		if policy.Action == ActionAllow {
			hasAllow = true
		}
		if !policy.SourceCIDR.Contains(address) {
			continue
		}
		if policy.Action == ActionDeny {
			return false
		}
		if policy.Action == ActionAllow {
			matchedAllow = true
		}
	}
	if matchedAllow {
		return true
	}
	return !hasAllow
}
