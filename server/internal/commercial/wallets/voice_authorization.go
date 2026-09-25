package wallets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

// ManagedCallAuthorization requests a prepaid hold for one managed call.
// The pricing quote is a customer tariff derived from a verified wholesale
// carrier rate, not the raw wholesale rate itself. The runtime must enforce
// MaximumSeconds independently before any live carrier origination.
type ManagedCallAuthorization struct {
	OrganizationID uuid.UUID
	CallID         uuid.UUID
	Quote          pricing.Quote
	MaximumSeconds int64
	ExpiresAt      time.Time
}

// AuthorizeManagedCall holds the highest customer-rated amount possible within
// the configured call duration. The authorization is not a fixed-duration
// product: the final capture uses only actual rated usage.
func (s *Service) AuthorizeManagedCall(
	ctx context.Context,
	authorization ManagedCallAuthorization,
) (sqlc.WalletReservation, error) {
	if authorization.OrganizationID == uuid.Nil || authorization.CallID == uuid.Nil {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"managed call authorization requires organization and call IDs",
		)
	}
	if authorization.MaximumSeconds <= 0 {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"managed call maximum duration must be positive",
		)
	}
	rating, err := authorization.Quote.Rate(authorization.MaximumSeconds)
	if err != nil {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"rate managed call authorization: %w",
			err,
		)
	}
	reservation, err := s.Reserve(ctx, ReserveRequest{
		OrganizationID: authorization.OrganizationID,
		Currency:       authorization.Quote.Currency,
		AmountMinor:    rating.CustomerAmountMinor,
		OperationType:  "managed_call",
		OperationID:    authorization.CallID.String(),
		ExpiresAt:      authorization.ExpiresAt,
	})
	if err != nil {
		return sqlc.WalletReservation{}, err
	}
	// A replay cannot reauthorize a captured, released, or expired hold.
	if reservation.Status != "active" ||
		!reservation.ExpiresAt.Time.After(time.Now()) {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"managed call has no active prepaid authorization",
		)
	}
	return reservation, nil
}

// ManagedCallSettlement is an internally sourced, final rated call record.
// The producer must independently authenticate and match BillingEvidence,
// OrganizationID, CallID, and the immutable authorization quote before calling
// this method. It must not accept an arbitrary customer-submitted quote.
type ManagedCallSettlement struct {
	OrganizationID  uuid.UUID
	CallID          uuid.UUID
	Quote           pricing.Quote
	ActualSeconds   int64
	BillingEvidence string
}

// GetManagedCallReservation finds the hold even after its expiry. An expiry
// must never be mistaken for proof that the carrier stopped billing.
func (s *Service) GetManagedCallReservation(
	ctx context.Context,
	organizationID uuid.UUID,
	callID uuid.UUID,
) (sqlc.WalletReservation, error) {
	if organizationID == uuid.Nil || callID == uuid.Nil {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"managed call reservation requires organization and call IDs",
		)
	}
	if s == nil || s.repo == nil || !s.repo.Available() {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"managed call reservation storage is unavailable",
		)
	}
	return s.repo.ReservationByOperation(
		ctx,
		organizationID,
		"managed_call",
		callID.String(),
	)
}

// SettleManagedCall rates actual usage from the customer tariff rather than
// accepting a precomputed debit amount. The wallet capture remains idempotent
// by logical call ID. A trusted caller must persist and verify the original
// quote and final carrier CDR before invoking this method in production.
func (s *Service) SettleManagedCall(
	ctx context.Context,
	settlement ManagedCallSettlement,
) (ReservationResult, error) {
	if settlement.OrganizationID == uuid.Nil || settlement.CallID == uuid.Nil {
		return ReservationResult{}, fmt.Errorf(
			"managed call settlement requires organization and call IDs",
		)
	}
	if strings.TrimSpace(settlement.BillingEvidence) == "" {
		return ReservationResult{}, fmt.Errorf(
			"managed call settlement requires final billing evidence",
		)
	}
	rating, err := settlement.Quote.Rate(settlement.ActualSeconds)
	if err != nil {
		return ReservationResult{}, fmt.Errorf(
			"rate managed call usage: %w",
			err,
		)
	}
	reservation, err := s.GetManagedCallReservation(
		ctx,
		settlement.OrganizationID,
		settlement.CallID,
	)
	if err != nil {
		return ReservationResult{}, err
	}
	if rating.CustomerAmountMinor > reservation.AmountMinor {
		return ReservationResult{}, fmt.Errorf(
			"managed call final charge exceeds prepaid reservation",
		)
	}
	if rating.CustomerAmountMinor == 0 {
		released, releaseErr := s.release(
			ctx,
			settlement.OrganizationID,
			reservation.ID,
			"released",
			true,
		)
		if releaseErr != nil {
			return ReservationResult{}, releaseErr
		}
		return ReservationResult{
			Reservation: released,
		}, nil
	}
	return s.Capture(ctx, CaptureRequest{
		OrganizationID: settlement.OrganizationID,
		ReservationID:  reservation.ID,
		AmountMinor:    rating.CustomerAmountMinor,
		Reason:         "managed_call",
		ReferenceType:  "managed_call",
		ReferenceID:    settlement.CallID,
	})
}
