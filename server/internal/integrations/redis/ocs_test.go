package redis

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateOCSWalletSeed(t *testing.T) {
	t.Parallel()

	seed := OCSWalletSeed{
		OrganizationID: uuid.New(),
		WalletID:       uuid.New(),
		Currency:       "USD",
		Status:         "active",
		BalanceMicros:  1_000_000,
		ReservedMicros: 100_000,
		Version:        4,
	}

	if err := validateOCSWalletSeed(seed); err != nil {
		t.Fatalf(
			"validate wallet seed: %v",
			err,
		)
	}

	seed.ReservedMicros = seed.BalanceMicros + 1
	if err := validateOCSWalletSeed(seed); err == nil {
		t.Fatal("expected invalid reserved balance to fail")
	}
}

func TestValidateOCSReserveRequest(t *testing.T) {
	t.Parallel()

	req := OCSReserveRequest{
		OrganizationID: uuid.New(),
		WalletID:       uuid.New(),
		Currency:       "USD",
		ChargeID:       uuid.New(),
		OperationID:    uuid.New(),
		AmountMicros:   20_000,
		ChargingMode:   "rolling",
		OccurredAt:     time.Now().UTC(),
	}

	if err := validateOCSReserveRequest(req); err != nil {
		t.Fatalf(
			"validate reserve request: %v",
			err,
		)
	}

	req.AmountMicros = 0
	if err := validateOCSReserveRequest(req); err == nil {
		t.Fatal("expected zero reserve amount to fail")
	}
}

func TestParseOCSResult(t *testing.T) {
	t.Parallel()

	value := []any{
		"ok",
		"1-0",
		int64(7),
		int64(3),
		int64(900_000),
		int64(40_000),
		int64(60_000),
		int64(20_000),
		int64(40_000),
		"active",
	}

	result, err := parseOCSResult(value)
	if err != nil {
		t.Fatalf(
			"parse OCS result: %v",
			err,
		)
	}

	if result.Code != "ok" {
		t.Fatalf(
			"unexpected result code: %s",
			result.Code,
		)
	}
	if result.WalletVersion != 7 {
		t.Fatalf(
			"unexpected wallet version: %d",
			result.WalletVersion,
		)
	}
	if result.ChargeSequence != 3 {
		t.Fatalf(
			"unexpected charge sequence: %d",
			result.ChargeSequence,
		)
	}
	if result.ChargeReservedMicros != 40_000 {
		t.Fatalf(
			"unexpected reserved amount: %d",
			result.ChargeReservedMicros,
		)
	}
}
