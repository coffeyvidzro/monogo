package settlement

import (
	"math"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
)

func TestValidateWalletProjection(t *testing.T) {
	t.Parallel()

	event := redisintegration.OCSEvent{
		WalletID:            uuid.New(),
		WalletVersion:       3,
		BalanceDeltaMicros:  -200,
		ReservedDeltaMicros: -200,
		BalanceAfterMicros:  800,
		ReservedAfterMicros: 100,
	}
	wallet := sqlc.Wallet{
		BalanceMicros:  1_000,
		ReservedMicros: 300,
		OcsVersion:     2,
	}
	if err := validateWalletProjection(wallet, event); err != nil {
		t.Fatalf(
			"validate projection: %v",
			err,
		)
	}
}

func TestValidateWalletProjectionDetectsVersionGap(t *testing.T) {
	t.Parallel()

	event := redisintegration.OCSEvent{
		WalletID:            uuid.New(),
		WalletVersion:       5,
		BalanceAfterMicros:  1_000,
		ReservedAfterMicros: 0,
	}
	wallet := sqlc.Wallet{
		BalanceMicros:  1_000,
		ReservedMicros: 0,
		OcsVersion:     2,
	}
	if err := validateWalletProjection(wallet, event); err == nil {
		t.Fatal("expected wallet version gap")
	}
}

func TestSafeDeltaMatchesRejectsOverflow(t *testing.T) {
	t.Parallel()

	if safeDeltaMatches(math.MaxInt64, 1, math.MinInt64) {
		t.Fatal("expected positive overflow to fail")
	}
	if safeDeltaMatches(math.MinInt64, -1, math.MaxInt64) {
		t.Fatal("expected negative overflow to fail")
	}
}

func TestValidateChargeTransition(t *testing.T) {
	t.Parallel()

	charge := sqlc.Charge{
		AuthorizedMicros: 100_000,
		ConsumedMicros:   20_000,
		ReservedMicros:   80_000,
	}
	tests := []struct {
		name  string
		event redisintegration.OCSEvent
	}{
		{
			name: "consume",
			event: redisintegration.OCSEvent{
				EventType:              "consume",
				BalanceDeltaMicros:     -20_000,
				ReservedDeltaMicros:    -20_000,
				ChargeAuthorizedMicros: 100_000,
				ChargeConsumedMicros:   40_000,
				ChargeReservedMicros:   60_000,
			},
		},
		{
			name: "release",
			event: redisintegration.OCSEvent{
				EventType:              "release",
				ReservedDeltaMicros:    -30_000,
				ChargeAuthorizedMicros: 100_000,
				ChargeConsumedMicros:   20_000,
				ChargeReservedMicros:   50_000,
			},
		},
		{
			name: "finalize",
			event: redisintegration.OCSEvent{
				EventType:              "finalize",
				ReservedDeltaMicros:    -80_000,
				ChargeAuthorizedMicros: 100_000,
				ChargeConsumedMicros:   20_000,
				ChargeReservedMicros:   0,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := validateChargeTransition(charge, test.event); err != nil {
				t.Fatalf(
					"validate transition: %v",
					err,
				)
			}
		})
	}
}

func TestValidateChargeTransitionRejectsInconsistentResult(t *testing.T) {
	t.Parallel()

	charge := sqlc.Charge{
		AuthorizedMicros: 100_000,
		ConsumedMicros:   20_000,
		ReservedMicros:   80_000,
	}
	event := redisintegration.OCSEvent{
		EventType:              "consume",
		BalanceDeltaMicros:     -20_000,
		ReservedDeltaMicros:    -20_000,
		ChargeAuthorizedMicros: 100_000,
		ChargeConsumedMicros:   60_000,
		ChargeReservedMicros:   60_000,
	}

	if err := validateChargeTransition(charge, event); err == nil {
		t.Fatal("expected inconsistent charge result to fail")
	}
}

func TestValidateReserveAndDebitTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		charge sqlc.Charge
		event  redisintegration.OCSEvent
	}{
		{
			name: "reserve",
			charge: sqlc.Charge{
				AuthorizedMicros: 0,
				ConsumedMicros:   0,
				ReservedMicros:   0,
			},
			event: redisintegration.OCSEvent{
				EventType:              "reserve",
				ReservedDeltaMicros:    100_000,
				ChargeAuthorizedMicros: 100_000,
				ChargeConsumedMicros:   0,
				ChargeReservedMicros:   100_000,
			},
		},
		{
			name: "debit",
			charge: sqlc.Charge{
				AuthorizedMicros: 0,
				ConsumedMicros:   0,
				ReservedMicros:   0,
			},
			event: redisintegration.OCSEvent{
				EventType:              "debit",
				BalanceDeltaMicros:     -100_000,
				ChargeAuthorizedMicros: 100_000,
				ChargeConsumedMicros:   100_000,
				ChargeReservedMicros:   0,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := validateChargeTransition(test.charge, test.event); err != nil {
				t.Fatalf(
					"validate transition: %v",
					err,
				)
			}
		})
	}
}

func TestSamePersistedEventIncludesChargeResult(t *testing.T) {
	t.Parallel()

	chargeID := uuid.New()
	chargeSequence := int64(2)
	authorized := int64(100_000)
	consumed := int64(20_000)
	reserved := int64(80_000)
	status := "active"
	occurredAt := time.Now().UTC().Truncate(time.Millisecond)
	event := redisintegration.OCSEvent{
		OrganizationID:         uuid.New(),
		WalletID:               uuid.New(),
		ChargeID:               &chargeID,
		OperationID:            uuid.New(),
		WalletVersion:          2,
		ChargeSequence:         chargeSequence,
		EventType:              "consume",
		BalanceDeltaMicros:     -20_000,
		ReservedDeltaMicros:    -20_000,
		BalanceAfterMicros:     980_000,
		ReservedAfterMicros:    80_000,
		ChargeAuthorizedMicros: authorized,
		ChargeConsumedMicros:   consumed,
		ChargeReservedMicros:   reserved,
		ChargeStatus:           status,
		OccurredAt:             occurredAt,
	}
	existing := sqlc.WalletEvent{
		WalletID:                    event.WalletID,
		OrganizationID:              event.OrganizationID,
		ChargeID:                    event.ChargeID,
		OperationID:                 event.OperationID,
		WalletVersion:               event.WalletVersion,
		ChargeSequence:              &chargeSequence,
		EventType:                   event.EventType,
		BalanceDeltaMicros:          event.BalanceDeltaMicros,
		ReservedDeltaMicros:         event.ReservedDeltaMicros,
		BalanceAfterMicros:          event.BalanceAfterMicros,
		ReservedAfterMicros:         event.ReservedAfterMicros,
		OccurredAt:                  pgconv.TimeToTimestamptz(occurredAt),
		ChargeAuthorizedAfterMicros: &authorized,
		ChargeConsumedAfterMicros:   &consumed,
		ChargeReservedAfterMicros:   &reserved,
		ChargeStatus:                &status,
	}

	if !samePersistedEvent(existing, event) {
		t.Fatal("expected identical event to be recognized as replay")
	}
	event.ChargeConsumedMicros++
	if samePersistedEvent(existing, event) {
		t.Fatal("expected changed charge result to be an integrity conflict")
	}
}
