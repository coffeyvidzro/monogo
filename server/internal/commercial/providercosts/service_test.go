package providercosts

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeRecordRequest(t *testing.T) {
	t.Parallel()
	req := RecordRequest{
		ProviderID:       uuid.New(),
		OperationID:      uuid.New(),
		ProviderRecordID: " invoice-1 ",
		Product:          "number_renewal",
		Currency:         "usd",
		AmountMicros:     250000,
		IncurredAt:       time.Date(2026, time.January, 1, 0, 0, 0, 0, time.FixedZone("test", 3600)),
	}
	if err := normalize(&req); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if req.ProviderRecordID != "invoice-1" || req.Currency != "USD" {
		t.Fatalf("request was not normalized: %+v", req)
	}
	if string(req.RawPayload) != "{}" || req.IncurredAt.Location() != time.UTC {
		t.Fatalf("evidence defaults were not normalized: %+v", req)
	}
}

func TestNormalizeRejectsUnsupportedProduct(t *testing.T) {
	t.Parallel()
	req := RecordRequest{
		ProviderID:       uuid.New(),
		OperationID:      uuid.New(),
		ProviderRecordID: "invoice-1",
		Product:          "unknown",
		Currency:         "USD",
		IncurredAt:       time.Now(),
	}
	if err := normalize(&req); err == nil {
		t.Fatal("expected unsupported product to be rejected")
	}
}
