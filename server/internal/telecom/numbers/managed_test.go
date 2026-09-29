package numbers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/google/uuid"
)

func TestNormalizeManagedPurchaseRequiresTenantAndIdempotency(t *testing.T) {
	valid := ManagedPurchaseRequest{Number: " +12125551234 ", CountryCode: " us "}
	if err := normalizeManagedPurchase(uuid.New(), "purchase-1", &valid); err != nil {
		t.Fatal(err)
	}
	if valid.Number != "+12125551234" || valid.CountryCode != "US" {
		t.Fatalf("unexpected normalization: %+v", valid)
	}
	for _, tc := range []struct {
		org uuid.UUID
		key string
		req ManagedPurchaseRequest
	}{
		{uuid.Nil, "key", valid}, {uuid.New(), "", valid},
		{uuid.New(), "key", ManagedPurchaseRequest{Number: "2125551234", CountryCode: "US"}},
		{uuid.New(), "key", ManagedPurchaseRequest{Number: "+12125551234", CountryCode: "USA"}},
	} {
		if err := normalizeManagedPurchase(tc.org, tc.key, &tc.req); err == nil {
			t.Fatalf("expected rejection for %+v", tc)
		}
	}
}

func TestManagedNumberOperationIDIsStablePerOrder(t *testing.T) {
	orderID := uuid.New()
	first := managedNumberOperationID(orderID)
	second := managedNumberOperationID(orderID)
	other := managedNumberOperationID(uuid.New())

	if first != second {
		t.Fatalf("replayed operation id = %s, want %s", second, first)
	}
	if first == other {
		t.Fatal("different orders produced the same billing operation id")
	}
}

func TestAvailableDIDSKUIsResolvedFromPrivateRelationship(t *testing.T) {
	var available didww.AvailableDID
	available.ID = "available-1"
	available.Relationships = map[string]json.RawMessage{"sku": json.RawMessage(`{"data":{"type":"skus","id":"sku-1"}}`)}
	sku, err := availableDIDSKU(available)
	if err != nil || sku != "sku-1" {
		t.Fatalf("availableDIDSKU = %q, %v", sku, err)
	}
	available.Relationships = nil
	if _, err := availableDIDSKU(available); err == nil {
		t.Fatal("expected missing SKU to fail closed")
	}
}

func TestDIDUsesOnlyConfiguredVoiceInTrunk(t *testing.T) {
	did := didww.DID{Relationships: map[string]didww.Relationship{
		"voice_in_trunk": {Data: &didww.ResourceIdentifier{Type: "voice_in_trunks", ID: "trusted"}},
	}}
	if !didUsesTrunk(did, "trusted") {
		t.Fatal("expected configured trunk to match")
	}
	if didUsesTrunk(did, "other") {
		t.Fatal("unexpected trunk matched")
	}
	relationship := did.Relationships["voice_in_trunk"]
	relationship.Data.Type = "voice_in_trunk_groups"
	did.Relationships["voice_in_trunk"] = relationship
	if didUsesTrunk(did, "trusted") {
		t.Fatal("wrong relationship type matched")
	}
}

func TestRenewalRetryDelayIsCapped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		attempt int32
		want    time.Duration
	}{
		{
			attempt: 1,
			want:    time.Hour,
		},
		{
			attempt: 2,
			want:    2 * time.Hour,
		},
		{
			attempt: 10,
			want:    24 * time.Hour,
		},
	}
	for _, testCase := range cases {
		if got := renewalRetryDelay(testCase.attempt); got != testCase.want {
			t.Fatalf("attempt %d: got %s, want %s", testCase.attempt, got, testCase.want)
		}
	}
}

func TestDecimalMicros(t *testing.T) {
	t.Parallel()
	cases := []struct {
		value string
		want  int64
	}{
		{
			value: "0",
			want:  0,
		},
		{
			value: "12.34",
			want:  12_340_000,
		},
		{
			value: "0.000001",
			want:  1,
		},
	}
	for _, testCase := range cases {
		got, err := decimalMicros(testCase.value)
		if err != nil {
			t.Fatalf("parse %q: %v", testCase.value, err)
		}
		if got != testCase.want {
			t.Fatalf("parse %q: got %d, want %d", testCase.value, got, testCase.want)
		}
	}
	for _, value := range []string{
		"-1",
		"1.0000001",
		"not-money",
		"",
	} {
		if _, err := decimalMicros(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
