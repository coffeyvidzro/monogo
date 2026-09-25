package wallets

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

// ManagedCallAuthorization defines a bounded, prepaid liability for one call.
// MaximumMinutes must be enforced by the media runtime before this can
// authorize a live managed call.
type ManagedCallAuthorization struct {
	OrganizationID uuid.UUID
	CallID         uuid.UUID
	Currency       string
	RateMicros     int64
	MaximumMinutes int64
	ExpiresAt      time.Time
}

// ManagedCallReserveMinor converts an upper bound in currency micros to minor
// units, rounding up rather than permitting fractional unpaid exposure.
// The rate applies to a whole minute; this is an upper-bound reservation,
// not a substitute for a carrier's verified customer tariff or billing rules.
func ManagedCallReserveMinor(rateMicros, maximumMinutes int64) (int64, error) {
	if rateMicros <= 0 || maximumMinutes <= 0 {
		return 0, fmt.Errorf("managed call rate and maximum duration must be positive")
	}
	if rateMicros > math.MaxInt64/maximumMinutes {
		return 0, fmt.Errorf("managed call authorization exceeds supported amount")
	}
	exposureMicros := rateMicros * maximumMinutes
	const microsPerMinor = int64(10_000)
	minor := exposureMicros / microsPerMinor
	if exposureMicros%microsPerMinor != 0 {
		minor++
	}
	if minor <= 0 {
		return 0, fmt.Errorf("managed call authorization amount is invalid")
	}
	return minor, nil
}

// AuthorizeManagedCall atomically holds a bounded amount in the customer's
// prepaid wallet using the existing idempotent reservation service.
// Live-call enablement additionally requires enforced duration, reservation
// renewal, terminal-event settlement, and uncertain-outcome recovery.
func (s *Service) AuthorizeManagedCall(
	ctx context.Context,
	authorization ManagedCallAuthorization,
) (sqlc.WalletReservation, error) {
	if authorization.OrganizationID == uuid.Nil || authorization.CallID == uuid.Nil {
		return sqlc.WalletReservation{}, fmt.Errorf("managed call authorization requires organization and call IDs")
	}
	if authorization.Currency != "USD" {
		return sqlc.WalletReservation{}, fmt.Errorf("managed call authorization requires a USD wallet")
	}
	minor, err := ManagedCallReserveMinor(
		authorization.RateMicros,
		authorization.MaximumMinutes,
	)
	if err != nil {
		return sqlc.WalletReservation{}, err
	}
	reservation, err := s.Reserve(ctx, ReserveRequest{
		OrganizationID: authorization.OrganizationID,
		Currency:       authorization.Currency,
		AmountMinor:    minor,
		OperationType:  "managed_call",
		OperationID:    authorization.CallID.String(),
		ExpiresAt:      authorization.ExpiresAt,
	})
	if err != nil {
		return sqlc.WalletReservation{}, err
	}
	// An idempotent replay may return a previously captured or released
	// reservation. Only an active, unexpired hold is valid authorization.
	if reservation.Status != "active" || !reservation.ExpiresAt.Time.After(time.Now()) {
		return sqlc.WalletReservation{}, fmt.Errorf(
			"managed call has no active prepaid authorization",
		)
	}
	return reservation, nil
}

// ManagedCallSettlement is supplied only after a trusted carrier billing
// reconciliation has established the final, customer-rated charge or
// explicitly confirmed that the call has no billable usage. An unanswered or
// locally failed call is not sufficient evidence for a zero-charge release.
type ManagedCallSettlement struct {
	OrganizationID  uuid.UUID
	CallID          uuid.UUID
	AmountMinor     int64
	BillingEvidence string
}

// GetManagedCallReservation finds the durable hold by the logical call ID.
// It can be called by a recovery worker even after the authorization expires.
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

// SettleManagedCall captures an externally verified final amount, or releases
// an explicitly confirmed zero-charge call. The call ID is the immutable
// financial reference: duplicate callbacks cannot create a second debit.
// Any amount above the reserved cap fails closed for manual reconciliation.
func (s *Service) SettleManagedCall(
	ctx context.Context,
	settlement ManagedCallSettlement,
) (ReservationResult, error) {
	if settlement.OrganizationID == uuid.Nil || settlement.CallID == uuid.Nil {
		return ReservationResult{}, fmt.Errorf(
			"managed call settlement requires organization and call IDs",
		)
	}
	if settlement.AmountMinor < 0 || strings.TrimSpace(settlement.BillingEvidence) == "" {
		return ReservationResult{}, fmt.Errorf(
			"managed call settlement requires a verified final billing reference and nonnegative amount",
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
	if settlement.AmountMinor > reservation.AmountMinor {
		return ReservationResult{}, fmt.Errorf(
			"managed call final charge exceeds prepaid reservation",
		)
	}
	if settlement.AmountMinor == 0 {
		released, err := s.release(
			ctx,
			settlement.OrganizationID,
			reservation.ID,
			"released",
			true,
		)
		if err != nil {
			return ReservationResult{}, err
		}
		return ReservationResult{
			Reservation: released,
		}, nil
	}
	return s.Capture(ctx, CaptureRequest{
		OrganizationID: settlement.OrganizationID,
		ReservationID:  reservation.ID,
		AmountMinor:    settlement.AmountMinor,
		Reason:         "managed_call",
		ReferenceType:  "managed_call",
		ReferenceID:    settlement.CallID,
	})
}
