package authorization

import (
	"context"
	"math"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type Service struct {
	pricing *pricing.Service
	wallets *wallets.Service
}

func NewService(
	pricingService *pricing.Service,
	walletService *wallets.Service,
) *Service {
	if pricingService == nil {
		panic("commercial authorization: pricing service is required")
	}
	if walletService == nil {
		panic("commercial authorization: wallet service is required")
	}

	return &Service{
		pricing: pricingService,
		wallets: walletService,
	}
}

// AuthorizeCall fails closed for managed traffic. BYOC traffic is explicitly
// non-billable and therefore never requires a Leamout rate or wallet balance.
func (s *Service) AuthorizeCall(
	ctx context.Context,
	req CallRequest,
) (Decision, error) {
	if err := normalizeCallRequest(&req); err != nil {
		return Decision{}, apperror.NewBadRequest(err.Error())
	}
	if req.ProvisioningMode == ProvisioningBYOC {
		return Decision{
			OperationID: req.OperationID,
			Billable:    false,
		}, nil
	}

	rate, err := s.pricing.Resolve(
		ctx,
		pricing.ResolveRequest{
			OrganizationID:    req.OrganizationID,
			DestinationDigits: req.Destination,
			Direction:         req.Direction,
			Currency:          CurrencyUSD,
			ResolvedAt:        req.RequestedAt,
		},
	)
	if err != nil {
		return Decision{}, err
	}

	amount, err := AmountForSeconds(rate.RateMicros, req.MinimumSeconds)
	if err != nil {
		return Decision{}, apperror.NewBadRequest(err.Error())
	}
	wallet, err := s.wallets.Get(ctx, req.OrganizationID)
	if err != nil {
		return Decision{}, err
	}
	if wallet.Currency != rate.Currency {
		return Decision{}, apperror.NewConflict("wallet and carrier rate currencies do not match")
	}
	if wallet.BalanceMicros < amount {
		return Decision{}, apperror.NewPaymentRequired("insufficient wallet balance")
	}

	currency := rate.Currency
	rateID := rate.ID
	return Decision{
		OperationID:            req.OperationID,
		Billable:               true,
		Currency:               &currency,
		RateID:                 &rateID,
		RateMicros:             rate.RateMicros,
		AuthorizedAmountMicros: amount,
	}, nil
}

// CaptureCall debits metered usage through the wallet's atomic, idempotent
// movement path. Callers must use the same operation ID used for authorization.
func (s *Service) CaptureCall(
	ctx context.Context,
	req CaptureRequest,
) (*wallets.LedgerEntry, error) {
	if err := validateCaptureRequest(req); err != nil {
		return nil, apperror.NewBadRequest(err.Error())
	}
	if !req.Decision.Billable {
		return nil, nil
	}

	amount, err := AmountForSeconds(req.Decision.RateMicros, req.BillableSeconds)
	if err != nil {
		return nil, apperror.NewBadRequest(err.Error())
	}
	if amount == 0 {
		return nil, nil
	}
	referenceType := "call"
	entry, err := s.wallets.Debit(
		ctx,
		wallets.MovementRequest{
			OrganizationID: req.OrganizationID,
			OperationID:    req.OperationID,
			AmountMicros:   amount,
			Reason:         "managed_call_usage",
			ReferenceType:  &referenceType,
			ReferenceID:    &req.ReferenceID,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// AmountForSeconds prices per-minute rates using whole started minutes. This
// prevents sub-minute managed usage from escaping billing.
func AmountForSeconds(rateMicros int64, seconds int64) (int64, error) {
	if rateMicros < 0 || seconds < 0 {
		return 0, ErrInvalidInput
	}
	if rateMicros == 0 || seconds == 0 {
		return 0, nil
	}
	minutes := seconds / 60
	if seconds%60 != 0 {
		minutes++
	}
	if minutes > math.MaxInt64/rateMicros {
		return 0, ErrInvalidInput
	}

	return minutes * rateMicros, nil
}

func normalizeCallRequest(req *CallRequest) error {
	req.Destination = strings.TrimPrefix(strings.TrimSpace(req.Destination), "+")
	req.Direction = strings.ToLower(strings.TrimSpace(req.Direction))
	req.ProvisioningMode = strings.ToLower(strings.TrimSpace(req.ProvisioningMode))
	if req.OrganizationID == uuid.Nil || req.OperationID == uuid.Nil || req.MinimumSeconds <= 0 {
		return ErrInvalidInput
	}
	if req.ProvisioningMode != ProvisioningBYOC && req.ProvisioningMode != ProvisioningManaged {
		return ErrInvalidInput
	}
	if req.Direction != pricing.DirectionInbound && req.Direction != pricing.DirectionOutbound {
		return ErrInvalidInput
	}
	if req.Destination == "" {
		return ErrInvalidInput
	}
	for _, digit := range req.Destination {
		if digit < '0' || digit > '9' {
			return ErrInvalidInput
		}
	}

	return nil
}

func validateCaptureRequest(req CaptureRequest) error {
	if req.OrganizationID == uuid.Nil || req.OperationID == uuid.Nil || req.ReferenceID == uuid.Nil {
		return ErrInvalidInput
	}
	if req.Decision.OperationID != req.OperationID || req.BillableSeconds < 0 {
		return ErrInvalidInput
	}
	if req.Decision.Billable && (req.Decision.Currency == nil || req.Decision.RateID == nil) {
		return ErrInvalidInput
	}

	return nil
}
