package usage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateRecord(t *testing.T) {
	valid := RecordRequest{
		OrganizationID: uuid.New(),
		MeterID: uuid.New(),
		Quantity: 1,
		SourceType: "voice_call",
		SourceID: "call-123",
		IdempotencyKey: "voice_call:call-123",
		OccurredAt: time.Now().UTC(),
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
