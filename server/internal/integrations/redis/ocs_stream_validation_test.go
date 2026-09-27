package redis

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateOCSEventRejectsMalformedFinancialState(t *testing.T) {
	t.Parallel()

	chargeID := uuid.New()
	valid := OCSEvent{
		StreamID:               "1000-0",
		OrganizationID:         uuid.New(),
		WalletID:               uuid.New(),
		ChargeID:               &chargeID,
		OperationID:            uuid.New(),
		WalletVersion:          1,
		ChargeSequence:         1,
		EventType:              "reserve",
		BalanceDeltaMicros:     0,
		ReservedDeltaMicros:    100,
		BalanceAfterMicros:     1_000,
		ReservedAfterMicros:    100,
		ChargeAuthorizedMicros: 100,
		ChargeConsumedMicros:   0,
		ChargeReservedMicros:   100,
		ChargeStatus:           "active",
		OccurredAt:             time.Now().UTC(),
	}

	tests := []struct {
		name   string
		mutate func(*OCSEvent)
	}{
		{
			name: "missing organization",
			mutate: func(event *OCSEvent) {
				event.OrganizationID = uuid.Nil
			},
		},
		{
			name: "non-positive wallet version",
			mutate: func(event *OCSEvent) {
				event.WalletVersion = 0
			},
		},
		{
			name: "negative resulting balance",
			mutate: func(event *OCSEvent) {
				event.BalanceAfterMicros = -1
			},
		},
		{
			name: "reservation exceeds balance",
			mutate: func(event *OCSEvent) {
				event.ReservedAfterMicros = 1_001
			},
		},
		{
			name: "missing charge id",
			mutate: func(event *OCSEvent) {
				event.ChargeID = nil
			},
		},
		{
			name: "invalid charge sequence",
			mutate: func(event *OCSEvent) {
				event.ChargeSequence = 0
			},
		},
		{
			name: "invalid occurrence time",
			mutate: func(event *OCSEvent) {
				event.OccurredAt = time.Time{}
			},
		},
		{
			name: "invalid event type",
			mutate: func(event *OCSEvent) {
				event.EventType = "unknown"
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			event := valid
			test.mutate(&event)
			if err := ValidateOCSEvent(event); err == nil {
				t.Fatal("expected malformed event to fail validation")
			}
		})
	}
}

func TestValidateOCSCreditRejectsChargeState(t *testing.T) {
	t.Parallel()

	chargeID := uuid.New()
	event := OCSEvent{
		StreamID:            "1000-0",
		OrganizationID:      uuid.New(),
		WalletID:            uuid.New(),
		ChargeID:            &chargeID,
		OperationID:         uuid.New(),
		WalletVersion:       1,
		EventType:           "credit",
		BalanceDeltaMicros:  100,
		BalanceAfterMicros:  1_100,
		ReservedAfterMicros: 0,
		OccurredAt:          time.Now().UTC(),
	}

	if err := ValidateOCSEvent(event); err == nil {
		t.Fatal("expected credit charge state to fail validation")
	}
}
