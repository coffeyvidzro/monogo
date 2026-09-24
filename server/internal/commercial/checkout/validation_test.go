package checkout

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateCreate(t *testing.T) {
	req := CreateRequest{
		OrganizationID: uuid.New(), WalletID: uuid.New(), AmountMinor: 100,
		Currency: " usd ", IdempotencyKey: " repeat ", ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := validateCreate(&req); err != nil {
		t.Fatal(err)
	}
	if req.Currency != "USD" || req.IdempotencyKey != "repeat" {
		t.Fatalf("unexpected normalized checkout: %+v", req)
	}
	first := requestHash(req)
	req.AmountMinor++
	if first == requestHash(req) {
		t.Fatal("checkout amount must affect idempotency hash")
	}
	req.AmountMinor = 0
	if err := validateCreate(&req); err == nil {
		t.Fatal("zero checkout amount was accepted")
	}
}
