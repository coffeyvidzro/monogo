package redis

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func ValidateOCSEvent(event OCSEvent) error {
	if event.StreamID == "" {
		return fmt.Errorf("OCS stream id is required")
	}
	if event.OrganizationID == uuid.Nil ||
		event.WalletID == uuid.Nil ||
		event.OperationID == uuid.Nil {
		return fmt.Errorf("OCS organization, wallet, and operation are required")
	}
	if event.WalletVersion <= 0 {
		return fmt.Errorf("OCS wallet version must be positive")
	}
	if event.BalanceAfterMicros < 0 || event.ReservedAfterMicros < 0 {
		return fmt.Errorf("OCS resulting wallet values cannot be negative")
	}
	if event.ReservedAfterMicros > event.BalanceAfterMicros {
		return fmt.Errorf("OCS resulting reservation exceeds balance")
	}
	if event.OccurredAt.IsZero() || event.OccurredAt.UnixMilli() <= 0 {
		return fmt.Errorf("OCS occurrence time is invalid")
	}
	if event.OccurredAt.After(time.Now().UTC().Add(5 * time.Minute)) {
		return fmt.Errorf("OCS occurrence time is unreasonably far in the future")
	}

	switch event.EventType {
	case "credit":
		return validateCreditEvent(event)
	case "reserve", "consume", "release", "debit":
		if event.ChargeStatus != "active" {
			return fmt.Errorf("%s event requires an active charge", event.EventType)
		}
	case "finalize":
		switch event.ChargeStatus {
		case "completed", "failed", "cancelled":
		default:
			return fmt.Errorf("finalize event has invalid charge status")
		}
	default:
		return fmt.Errorf("unsupported OCS event type %q", event.EventType)
	}

	return validateChargeEvent(event)
}

func validateCreditEvent(event OCSEvent) error {
	if event.ChargeID != nil ||
		event.ChargeSequence != 0 ||
		event.ChargeAuthorizedMicros != 0 ||
		event.ChargeConsumedMicros != 0 ||
		event.ChargeReservedMicros != 0 ||
		event.ChargeStatus != "" {
		return fmt.Errorf("credit event contains incompatible charge state")
	}
	if event.BalanceDeltaMicros <= 0 || event.ReservedDeltaMicros != 0 {
		return fmt.Errorf("credit event contains invalid wallet deltas")
	}

	return nil
}

func validateChargeEvent(event OCSEvent) error {
	if event.ChargeID == nil || *event.ChargeID == uuid.Nil {
		return fmt.Errorf("charge event requires a charge id")
	}
	if event.ChargeSequence <= 0 {
		return fmt.Errorf("charge event sequence must be positive")
	}
	if event.ChargeAuthorizedMicros < 0 ||
		event.ChargeConsumedMicros < 0 ||
		event.ChargeReservedMicros < 0 {
		return fmt.Errorf("OCS resulting charge values cannot be negative")
	}
	if event.ChargeConsumedMicros > event.ChargeAuthorizedMicros-event.ChargeReservedMicros {
		return fmt.Errorf("OCS resulting charge values exceed authorization")
	}
	if event.ChargeStatus != "active" && event.ChargeReservedMicros != 0 {
		return fmt.Errorf("terminal OCS charge retains a reservation")
	}

	switch event.EventType {
	case "reserve":
		if event.BalanceDeltaMicros != 0 || event.ReservedDeltaMicros <= 0 {
			return fmt.Errorf("reserve event contains invalid wallet deltas")
		}
	case "consume":
		if event.BalanceDeltaMicros >= 0 ||
			event.ReservedDeltaMicros >= 0 ||
			event.BalanceDeltaMicros != event.ReservedDeltaMicros {
			return fmt.Errorf("consume event contains invalid wallet deltas")
		}
	case "release":
		if event.BalanceDeltaMicros != 0 || event.ReservedDeltaMicros >= 0 {
			return fmt.Errorf("release event contains invalid wallet deltas")
		}
	case "debit":
		if event.BalanceDeltaMicros >= 0 || event.ReservedDeltaMicros != 0 {
			return fmt.Errorf("debit event contains invalid wallet deltas")
		}
	case "finalize":
		if event.BalanceDeltaMicros != 0 || event.ReservedDeltaMicros > 0 {
			return fmt.Errorf("finalize event contains invalid wallet deltas")
		}
	}

	return nil
}
