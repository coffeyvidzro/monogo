package providercharges

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeProviderChargeRequest(t *testing.T) {
	t.Parallel()
	req := RecordProviderChargeRequest{
		ProviderID:         uuid.New(),
		OperationID:        uuidPointer(uuid.New()),
		ProviderRecordType: "invoice_item",
		ProviderRecordID:   " invoice-1 ",
		Product:            "number_renewal",
		Currency:           "usd",
		AmountMicros:       250000,
		IncurredAt:         time.Date(2026, time.January, 1, 0, 0, 0, 0, time.FixedZone("test", 3600)),
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
	req := RecordProviderChargeRequest{
		ProviderID:         uuid.New(),
		OperationID:        uuidPointer(uuid.New()),
		ProviderRecordType: "invoice_item",
		ProviderRecordID:   "invoice-1",
		Product:            "unknown",
		Currency:           "USD",
		IncurredAt:         time.Now(),
	}
	if err := normalize(&req); err == nil {
		t.Fatal("expected unsupported product to be rejected")
	}
}

func uuidPointer(id uuid.UUID) *uuid.UUID {
	return &id
}

func TestNormalizeAllowsUnreconciledProviderCharge(t *testing.T) {
	t.Parallel()
	req := RecordProviderChargeRequest{
		ProviderID:         uuid.New(),
		ProviderRecordType: "order",
		ProviderRecordID:   "provider-order-1",
		Product:            "number_purchase",
		Currency:           "USD",
		AmountMicros:       100_000,
		IncurredAt:         time.Now(),
	}
	if err := normalize(&req); err != nil {
		t.Fatalf("normalize unreconciled charge: %v", err)
	}
	if req.OperationID != nil {
		t.Fatal("unreconciled charge unexpectedly gained an operation id")
	}
}

func TestNormalizeRejectsVoiceWithoutCDR(t *testing.T) {
	t.Parallel()
	req := RecordProviderChargeRequest{
		ProviderID:         uuid.New(),
		ProviderRecordType: "invoice_item",
		ProviderRecordID:   "voice-1",
		Product:            "voice",
		Currency:           "USD",
		AmountMicros:       100_000,
		IncurredAt:         time.Now(),
	}
	if err := normalize(&req); err == nil {
		t.Fatal("expected generic voice charge to require CDR ingestion")
	}
}
