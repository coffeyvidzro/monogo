package redis

import (
	"testing"
	"time"

	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

func TestDecodeOCSEvent(t *testing.T) {
	t.Parallel()

	organizationID := uuid.New()
	walletID := uuid.New()
	chargeID := uuid.New()
	operationID := uuid.New()
	message := redisv9.XMessage{
		ID: "1000-0",
		Values: map[string]any{
			"organization_id":                organizationID.String(),
			"wallet_id":                      walletID.String(),
			"charge_id":                      chargeID.String(),
			"operation_id":                   operationID.String(),
			"wallet_version":                 "4",
			"charge_sequence":                "2",
			"event_type":                     "consume",
			"balance_delta_micros":           "-250",
			"reserved_delta_micros":          "-250",
			"balance_after_micros":           "750",
			"reserved_after_micros":          "0",
			"charge_authorized_after_micros": "250",
			"charge_consumed_after_micros":   "250",
			"charge_reserved_after_micros":   "0",
			"charge_status":                  "active",
			"occurred_at_millis":             "1000",
		},
	}

	event, err := decodeOCSEvent(message)
	if err != nil {
		t.Fatalf(
			"decode event: %v",
			err,
		)
	}
	if event.ChargeID == nil || *event.ChargeID != chargeID {
		t.Fatalf("charge id = %v, want %s", event.ChargeID, chargeID)
	}
	if event.WalletVersion != 4 || event.ChargeSequence != 2 {
		t.Fatalf(
			"event versions = (%d, %d), want (4, 2)",
			event.WalletVersion,
			event.ChargeSequence,
		)
	}
	if !event.OccurredAt.Equal(time.UnixMilli(1_000).UTC()) {
		t.Fatalf(
			"occurred at = %s, want %s",
			event.OccurredAt,
			time.UnixMilli(1_000).UTC(),
		)
	}
}

func TestDecodeOCSEventRejectsInvalidChargeState(t *testing.T) {
	t.Parallel()

	message := redisv9.XMessage{
		ID: "1000-0",
		Values: map[string]any{
			"organization_id":                uuid.NewString(),
			"wallet_id":                      uuid.NewString(),
			"charge_id":                      uuid.NewString(),
			"operation_id":                   uuid.NewString(),
			"wallet_version":                 "1",
			"charge_sequence":                "1",
			"event_type":                     "finalize",
			"balance_delta_micros":           "0",
			"reserved_delta_micros":          "0",
			"balance_after_micros":           "0",
			"reserved_after_micros":          "0",
			"charge_authorized_after_micros": "1",
			"charge_consumed_after_micros":   "1",
			"charge_reserved_after_micros":   "0",
			"charge_status":                  "active",
			"occurred_at_millis":             "1000",
		},
	}

	if _, err := decodeOCSEvent(message); err == nil {
		t.Fatal("expected invalid final charge state to fail")
	}
}
