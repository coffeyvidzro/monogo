package charging

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateReserve(t *testing.T) {
	t.Parallel()

	req := ReserveRequest{
		OrganizationID: uuid.New(),
		ChargeID:       uuid.New(),
		OperationID:    uuid.New(),
		AmountMicros:   20_000,
		OccurredAt:     time.Now().UTC(),
	}

	if err := validateReserve(req); err != nil {
		t.Fatalf(
			"validate reserve: %v",
			err,
		)
	}

	req.AmountMicros = 0
	if err := validateReserve(req); err == nil {
		t.Fatal("expected zero reserve amount to fail")
	}
}

func TestValidateFinalize(t *testing.T) {
	t.Parallel()

	req := FinalizeRequest{
		OrganizationID: uuid.New(),
		ChargeID:       uuid.New(),
		OperationID:    uuid.New(),
		Status:         "completed",
		OccurredAt:     time.Now().UTC(),
	}

	if err := validateFinalize(req); err != nil {
		t.Fatalf(
			"validate finalize: %v",
			err,
		)
	}

	req.Status = "active"
	if err := validateFinalize(req); err == nil {
		t.Fatal("expected non-terminal status to fail")
	}
}
