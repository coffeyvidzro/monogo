package wallets

import (
	"context"
	"fmt"
	"math"
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
