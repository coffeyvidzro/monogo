package payments

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateAttempt(t *testing.T) {
	req := Attempt{OrganizationID: uuid.New(), CheckoutID: uuid.New(),
		Provider: " Stripe ", AttemptKey: " one ", AmountMinor: 500, Currency: " usd "}
	if err := validateAttempt(&req); err != nil {
		t.Fatal(err)
	}
	if req.Provider != "stripe" || req.Currency != "USD" || req.AttemptKey != "one" {
		t.Fatalf("unexpected normalization: %+v", req)
	}
	req.AmountMinor = 0
	if err := validateAttempt(&req); err == nil {
		t.Fatal("expected nonpositive amount to fail")
	}
}

func TestValidateSettlement(t *testing.T) {
	req := VerifiedSettlement{
		OrganizationID: uuid.New(), PaymentID: uuid.New(), Provider: " Stripe ",
		ProviderReference: " pi_123 ", AmountMinor: 2500, Currency: " usd ",
		VerifiedAt: time.Now().Add(-time.Minute),
	}
	if err := validateSettlement(&req); err != nil {
		t.Fatal(err)
	}
	if req.Provider != "stripe" || req.ProviderReference != "pi_123" || req.Currency != "USD" {
		t.Fatalf("unexpected normalization: %+v", req)
	}

	bad := req
	bad.VerifiedAt = time.Now().Add(10 * time.Minute)
	if err := validateSettlement(&bad); err == nil {
		t.Fatal("expected future verification to fail")
	}
	bad = req
	bad.ProviderReference = " "
	if err := validateSettlement(&bad); err == nil {
		t.Fatal("expected empty provider reference to fail")
	}
	bad = req
	bad.AmountMinor++
	bad.Currency = "US1"
	if err := validateSettlement(&bad); err == nil {
		t.Fatal("expected invalid currency to fail")
	}
}

func TestValidateEvent(t *testing.T) {
	req := Event{Provider: "paystack", ProviderEventID: "reference:charge.success",
		EventType: "charge.success", PayloadSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	if err := validateEvent(&req); err != nil {
		t.Fatal(err)
	}
	req.PayloadSHA256 = "not-a-digest"
	if err := validateEvent(&req); err == nil {
		t.Fatal("expected digest validation to fail")
	}
}
