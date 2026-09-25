package wallets

import (
	"context"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/google/uuid"
)

func testManagedCallQuote(t *testing.T) pricing.Quote {
	t.Helper()
	quote, err := pricing.QuoteFromWholesale(
		1_000_000,
		2_000,
		60,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	return quote
}

func TestManagedCallAuthorizationRatesMaximumCustomerExposure(t *testing.T) {
	quote := testManagedCallQuote(t)
	maximum, err := quote.Rate(120)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := quote.Rate(61)
	if err != nil {
		t.Fatal(err)
	}
	if maximum.CustomerAmountMinor != 240 || actual.CustomerAmountMinor != 240 {
		t.Fatalf("unexpected prepaid max or actual charges: max=%+v actual=%+v",
			maximum, actual)
	}
	shortCall, err := quote.Rate(10)
	if err != nil {
		t.Fatal(err)
	}
	if shortCall.CustomerAmountMinor != 120 {
		t.Fatalf("short call charged %d minor, want 120", shortCall.CustomerAmountMinor)
	}
}

func TestManagedCallAuthorizationRejectsUnpricedQuote(t *testing.T) {
	quote := testManagedCallQuote(t)
	quote.CustomerRateMicros = quote.WholesaleRateMicros
	service := (*Service)(nil)
	_, err := service.AuthorizeManagedCall(
		context.Background(),
		ManagedCallAuthorization{
			OrganizationID: uuid.New(),
			CallID:         uuid.New(),
			Quote:          quote,
			MaximumSeconds: 60,
			ExpiresAt:      time.Now().Add(time.Minute),
		},
	)
	if err == nil {
		t.Fatal("an unpriced or altered customer tariff was accepted")
	}
}

func TestManagedCallSettlementRequiresFinalBillingEvidence(t *testing.T) {
	service := (*Service)(nil)
	quote := testManagedCallQuote(t)
	tests := []struct {
		name       string
		settlement ManagedCallSettlement
	}{
		{
			name: "missing organization",
			settlement: ManagedCallSettlement{
				CallID:          uuid.New(),
				Quote:           quote,
				ActualSeconds:   60,
				BillingEvidence: "verified-cdr",
			},
		},
		{
			name: "missing call ID",
			settlement: ManagedCallSettlement{
				OrganizationID:  uuid.New(),
				Quote:           quote,
				ActualSeconds:   60,
				BillingEvidence: "verified-cdr",
			},
		},
		{
			name: "negative usage",
			settlement: ManagedCallSettlement{
				OrganizationID:  uuid.New(),
				CallID:          uuid.New(),
				Quote:           quote,
				ActualSeconds:   -1,
				BillingEvidence: "verified-cdr",
			},
		},
		{
			name: "unverified zero-charge outcome",
			settlement: ManagedCallSettlement{
				OrganizationID: uuid.New(),
				CallID:         uuid.New(),
				Quote:          quote,
			},
		},
		{
			name: "whitespace-only billing evidence",
			settlement: ManagedCallSettlement{
				OrganizationID:  uuid.New(),
				CallID:          uuid.New(),
				Quote:           quote,
				ActualSeconds:   60,
				BillingEvidence: "  ",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.SettleManagedCall(context.Background(), test.settlement)
			if err == nil {
				t.Fatal("invalid managed call settlement was accepted")
			}
		})
	}
}
