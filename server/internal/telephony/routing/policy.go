package routing

// Policy is the domain policy boundary for routing decisions. The default
// implementation currently validates the inbound route tuple already resolved
// by the deployment's SIP router. Capacity, health, and regional policy can be
// composed here without moving those concerns into calls.
type Policy interface {
	ValidateInbound(InboundRequest) error
}

type DefaultPolicy struct{}

func (DefaultPolicy) ValidateInbound(req InboundRequest) error {
	return validateInboundRequest(req)
}
