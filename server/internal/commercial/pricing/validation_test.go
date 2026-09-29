package pricing

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizeCreateVoiceRateRequiresPositiveRate(t *testing.T) {
	req := CreateVoiceRateRequest{
		DestinationPrefix: "1",
		Direction:         DirectionOutbound,
		Currency:          "USD",
		RateMicros:        0,
		EffectiveAt:       time.Now().UTC(),
	}

	if err := normalizeCreateVoiceRateRequest(&req); err == nil {
		t.Fatal("expected zero voice rate to be rejected")
	}
}

func TestNormalizeResolveProductRate(t *testing.T) {
	req := ResolveProductRateRequest{
		OrganizationID: uuid.New(),
		Product:        ProductNumberPurchase,
		Selector:       " us ",
		Currency:       " usd ",
	}

	if err := normalizeResolveProductRateRequest(&req); err != nil {
		t.Fatalf("normalize product rate: %v", err)
	}
	if req.Selector != "US" {
		t.Fatalf("selector = %q, want US", req.Selector)
	}
	if req.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", req.Currency)
	}
}

func TestNormalizeResolveProductRateRejectsArbitrarySelector(t *testing.T) {
	req := ResolveProductRateRequest{
		OrganizationID: uuid.New(),
		Product:        ProductSMSOutbound,
		Selector:       "NORTH_AMERICA",
		Currency:       "USD",
	}

	if err := normalizeResolveProductRateRequest(&req); err == nil {
		t.Fatal("expected arbitrary product selector to be rejected")
	}
}
