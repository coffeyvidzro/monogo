package payments

import (
	"testing"

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
