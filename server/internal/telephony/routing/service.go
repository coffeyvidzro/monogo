package routing

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo   *Repository
	policy Policy
}

func NewService(repo *Repository, policy Policy) *Service {
	if repo == nil {
		panic("routing: repository is required")
	}
	if policy == nil {
		policy = DefaultPolicy{}
	}
	return &Service{repo: repo, policy: policy}
}

// ResolveInbound revalidates the DID-derived organization, application, and
// trunk tuple before the call domain admits or persists an inbound call.
func (s *Service) ResolveInbound(
	ctx context.Context,
	req InboundRequest,
) (InboundDecision, error) {
	req = normalizeInboundRequest(req)
	if err := s.policy.ValidateInbound(req); err != nil {
		return InboundDecision{}, err
	}

	inbound, err := s.repo.GetInboundContext(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return InboundDecision{}, apperror.NewNotFound("no eligible inbound route")
	}
	if err != nil {
		return InboundDecision{}, apperror.NewInternal("resolve inbound route", err)
	}

	return InboundDecision{
		OrganizationID:      req.OrganizationID,
		VoiceAgentID:        req.VoiceAgentID,
		PhoneNumberID:       req.PhoneNumberID,
		VoiceAgentBindingID: req.VoiceAgentBindingID,
		TrunkID: req.TrunkID,
		CalledNumber:        req.CalledNumber,
		Limits:              inbound.Limits,
	}, nil
}

func (s *Service) ResolveOutbound(
	ctx context.Context,
	req OutboundRequest,
) (OutboundDecision, error) {
	req = normalizeOutboundRequest(req)
	if err := validateOutboundRequest(req); err != nil {
		return OutboundDecision{}, err
	}
	if req.TrunkID == nil {
		return OutboundDecision{}, apperror.NewBadRequest("trunk_id is required for outbound calls")
	}

	routes, err := s.repo.ResolveBYOCOutbound(ctx, req.OrganizationID, *req.TrunkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OutboundDecision{}, apperror.NewNotFound("no eligible outbound route")
	}
	if err != nil {
		return OutboundDecision{}, apperror.NewInternal("resolve outbound route", err)
	}
	return OutboundDecision{Routes: routes}, nil
}
