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
	return &Service{
		repo:   repo,
		policy: policy,
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
		CarrierConnectionID: req.CarrierConnectionID,
		CalledNumber:        req.CalledNumber,
		Limits:              limits,
	}, nil
}
