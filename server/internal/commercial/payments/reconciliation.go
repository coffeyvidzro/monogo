package payments

import (
	"context"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

// This module owns internal provider reconciliation, not customer-initiated
// verification. Provider submission and automated recovery remain separately
// gated until a durable attempt reference exists.

// VerifyAndSettle is an internal recovery/webhook entry point. It verifies
// the persisted provider reference, not a customer-provided payment status.
func (s *Service) VerifyAndSettle(
	ctx context.Context,
	organizationID, paymentID uuid.UUID,
) (SettlementResult, error) {
	if s == nil || !s.repo.Available() {
		return SettlementResult{}, apperror.NewServiceUnavailable("payment verification is not configured", nil)
	}
	payment, err := s.Get(ctx, organizationID, paymentID)
	if err != nil {
		return SettlementResult{}, err
	}
	if payment.ProviderReference == nil || *payment.ProviderReference == "" {
		return SettlementResult{}, apperror.NewConflict("payment has no persisted provider reference")
	}
	verifier := s.verifiers[payment.Provider]
	if verifier == nil {
		return SettlementResult{}, apperror.NewServiceUnavailable("payment provider verification is not configured", nil)
	}
	verification, err := verifier.Verify(ctx, RecoveryCandidate{
		OrganizationID:    organizationID,
		PaymentID:         paymentID,
		Provider:          payment.Provider,
		ProviderReference: *payment.ProviderReference,
		AmountMinor:       payment.AmountMinor,
		Currency:          payment.Currency,
	})
	if err != nil {
		return SettlementResult{}, apperror.NewServiceUnavailable("payment provider verification failed", err)
	}
	if !verification.Succeeded {
		return SettlementResult{}, apperror.NewConflict("payment is not verified as settled")
	}
	return s.Settle(ctx, VerifiedSettlement{
		OrganizationID:    organizationID,
		PaymentID:         paymentID,
		Provider:          payment.Provider,
		ProviderReference: *payment.ProviderReference,
		AmountMinor:       verification.AmountMinor,
		Currency:          verification.Currency,
		VerifiedAt:        verification.VerifiedAt,
	})
}
