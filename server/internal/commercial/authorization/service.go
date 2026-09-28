package authorization

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo    *Repository
	pricing *pricing.Service
	now     func() time.Time
}

func NewService(repo *Repository, pricingService *pricing.Service) *Service {
	if repo == nil {
		panic("commercial authorization: repository is required")
	}
	if pricingService == nil {
		panic("commercial authorization: pricing service is required")
	}
	return &Service{
		repo:    repo,
		pricing: pricingService,
		now:     time.Now,
	}
}

// AuthorizeManagedCall currently permits only explicitly free customer rates.
// Positive-rate calls remain fail-closed until atomic holds, incremental
// charging, and an exhaustion hangup path exist.
func (s *Service) AuthorizeManagedCall(ctx context.Context, req ManagedCallRequest) (CallAuthorization, error) {
	req.Destination = strings.TrimPrefix(strings.TrimSpace(req.Destination), "+")
	if req.CallID == uuid.Nil || req.OrganizationID == uuid.Nil || req.Destination == "" {
		return CallAuthorization{}, apperror.NewBadRequest("invalid managed call authorization")
	}
	if req.RequestedAt.IsZero() {
		req.RequestedAt = s.now().UTC()
	}
	rate, err := s.pricing.Resolve(ctx, pricing.ResolveRequest{
		OrganizationID:    req.OrganizationID,
		DestinationDigits: req.Destination,
		Direction:         req.Direction,
		Currency:          wallets.CurrencyUSD,
		ResolvedAt:        req.RequestedAt,
	})
	if err != nil {
		return CallAuthorization{}, err
	}
	if rate.RateMicros > 0 {
		return CallAuthorization{}, apperror.NewServiceUnavailable(
			"paid managed calls require prepaid wallet reservations",
			nil,
		)
	}
	authorization, err := s.repo.PersistFreeCallAuthorization(ctx, req.CallID, req.OrganizationID, rate, req.RequestedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CallAuthorization{}, apperror.NewConflict("call cannot be commercially authorized")
	}
	if err != nil {
		return CallAuthorization{}, apperror.NewInternal("persist call commercial authorization", err)
	}
	return authorization, nil
}
