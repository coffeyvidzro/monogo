package routing

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo                *Repository
	policy              Policy
	orchestrationPolicy OrchestrationPolicy
	now                 func() time.Time
}

func NewService(repo *Repository, policy Policy) *Service {
	if repo == nil {
		panic("routing: repository is required")
	}
	if policy == nil {
		policy = DefaultPolicy{}
	}
	return &Service{
		repo: repo, policy: policy,
		orchestrationPolicy: DefaultOrchestrationPolicy(),
		now:                 time.Now,
	}
}

// ResolveInbound revalidates the DID-derived organization, application, and
// carrier tuple before the call domain admits or persists an inbound call.
func (s *Service) ResolveInbound(
	ctx context.Context,
	req InboundRequest,
) (InboundDecision, error) {
	req = normalizeInboundRequest(req)
	if err := s.policy.ValidateInbound(req); err != nil {
		return InboundDecision{}, err
	}

	limits, err := s.repo.GetInboundContext(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return InboundDecision{}, apperror.NewNotFound("no eligible inbound route")
	}
	if err != nil {
		return InboundDecision{}, apperror.NewInternal("resolve inbound route", err)
	}

	return InboundDecision{
		OrganizationID:      req.OrganizationID,
		ApplicationID:       req.ApplicationID,
		PhoneNumberID:       req.PhoneNumberID,
		VoiceBindingID:      req.VoiceBindingID,
		CarrierConnectionID: req.CarrierConnectionID,
		CalledNumber:        req.CalledNumber,
		Limits:              limits,
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

	if req.TrunkID != nil {
		route, err := s.repo.ResolveBYOCOutbound(ctx, req.OrganizationID, *req.TrunkID)
		if errors.Is(err, pgx.ErrNoRows) {
			return OutboundDecision{}, apperror.NewNotFound("no eligible outbound route")
		}
		if err != nil {
			return OutboundDecision{}, apperror.NewInternal("resolve outbound route", err)
		}
		return OutboundDecision{Routes: []OutboundRoute{route}}, nil
	}

	destination, digits, err := managedDestination(req.Destination)
	if err != nil {
		return OutboundDecision{}, err
	}
	now := s.now().UTC()
	candidates, err := s.repo.ListManagedOutboundCandidates(ctx, digits, now)
	if err != nil {
		return OutboundDecision{}, apperror.NewInternal("load managed route candidates", err)
	}
	normalized := make([]CarrierCandidate, 0, len(candidates))
	byEndpoint := make(map[uuid.UUID]managedRouteCandidate, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, candidate.Candidate)
		byEndpoint[candidate.Candidate.EndpointID] = candidate
	}
	ranked, err := RankCarriers(now, s.orchestrationPolicy, normalized)
	if errors.Is(err, ErrNoEligibleCarrier) {
		return OutboundDecision{}, apperror.NewNotFound("no eligible outbound route")
	}
	if err != nil {
		return OutboundDecision{}, apperror.NewInternal("rank managed route candidates", err)
	}
	routes := make([]OutboundRoute, 0, len(ranked))
	for _, rankedCandidate := range ranked {
		route := byEndpoint[rankedCandidate.Candidate.EndpointID].Route
		route.ScoreMicros = rankedCandidate.Score
		routes = append(routes, route)
	}
	decisionID, err := s.repo.RecordManagedDecision(ctx, req.OrganizationID, destination, ranked, byEndpoint)
	if err != nil {
		return OutboundDecision{}, apperror.NewInternal("record managed routing decision", err)
	}
	return OutboundDecision{ID: decisionID, Routes: routes}, nil
}
