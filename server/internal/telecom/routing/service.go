package routing

import "context"

type Service struct {
	resolver *Resolver
}

func NewService(resolver *Resolver) *Service {
	return &Service{resolver: resolver}
}

func (s *Service) ResolveOutbound(
	ctx context.Context,
	req OutboundRequest,
) (OutboundDecision, error) {
	if err := validateOutboundRequest(req); err != nil {
		return OutboundDecision{}, err
	}
	return s.resolver.resolveOutbound(ctx, req)
}

func (s *Service) ResolveInbound(
	ctx context.Context,
	req InboundRequest,
) (InboundDecision, error) {
	sourceIP, err := validateInboundRequest(req)
	if err != nil {
		return InboundDecision{}, err
	}
	return s.resolver.resolveInbound(ctx, req, sourceIP)
}
