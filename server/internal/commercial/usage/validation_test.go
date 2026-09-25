package usage

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func TestValidateRecord(t *testing.T) {
	valid := RecordRequest{
		OrganizationID: uuid.New(),
		MeterID:        uuid.New(),
		Quantity:       1,
		SourceType:     "voice_call",
		SourceID:       "call-123",
		IdempotencyKey: "voice_call:call-123",
		OccurredAt:     time.Now().UTC(),
	}
	if err := validateRecord(&valid); err != nil {
		t.Fatal(err)
	}
	if string(valid.Dimensions) != "{}" {
		t.Fatalf("default dimensions = %q; want {}", valid.Dimensions)
	}
	bad := []RecordRequest{
		{OrganizationID: uuid.Nil, MeterID: valid.MeterID, Quantity: 1, SourceType: valid.SourceType, SourceID: valid.SourceID, IdempotencyKey: valid.IdempotencyKey, OccurredAt: valid.OccurredAt},
		{OrganizationID: valid.OrganizationID, MeterID: valid.MeterID, Quantity: 0, SourceType: valid.SourceType, SourceID: valid.SourceID, IdempotencyKey: valid.IdempotencyKey, OccurredAt: valid.OccurredAt},
		{OrganizationID: valid.OrganizationID, MeterID: valid.MeterID, Quantity: 1, SourceType: "Voice.Call", SourceID: valid.SourceID, IdempotencyKey: valid.IdempotencyKey, OccurredAt: valid.OccurredAt},
		{OrganizationID: valid.OrganizationID, MeterID: valid.MeterID, Quantity: 1, SourceType: valid.SourceType, SourceID: valid.SourceID, IdempotencyKey: valid.IdempotencyKey, Dimensions: json.RawMessage("[]"), OccurredAt: valid.OccurredAt},
		{OrganizationID: valid.OrganizationID, MeterID: valid.MeterID, Quantity: 1, SourceType: valid.SourceType, SourceID: valid.SourceID, IdempotencyKey: " invalid ", OccurredAt: valid.OccurredAt},
	}
	for _, req := range bad {
		if err := validateRecord(&req); err == nil {
			t.Fatalf("invalid observation accepted: %+v", req)
		}
	}
}

func TestValidCurrency(t *testing.T) {
	for _, value := range []string{"USD", "GHS", "EUR"} {
		if !validCurrency(value) {
			t.Fatalf("validCurrency(%q) = false", value)
		}
	}
	for _, value := range []string{"usd", "US", "US1", "USDD"} {
		if validCurrency(value) {
			t.Fatalf("validCurrency(%q) = true", value)
		}
	}
}

func TestChargeRejectsInvalidRatingBeforePersistence(t *testing.T) {
	request := ChargeRequest{Usage: RecordRequest{OrganizationID: uuid.New(), MeterID: uuid.New(), Quantity: 1,
		SourceType: "voice_call", SourceID: "call-1", IdempotencyKey: "call-1", OccurredAt: time.Now()},
		Currency: "USD", AmountMinor: 0}
	_, err := (&Service{}).Charge(t.Context(), request)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "BAD_REQUEST" {
		t.Fatalf("Charge() error = %v; want BAD_REQUEST", err)
	}
}
