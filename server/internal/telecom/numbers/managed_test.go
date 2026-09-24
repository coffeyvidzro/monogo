package numbers

import (
	"encoding/json"
	"testing"

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
