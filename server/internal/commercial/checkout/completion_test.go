package checkout

import (
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestValidateCompletion(t *testing.T) {
	checkoutID, walletID, paymentID, transactionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	current := sqlc.Checkout{
		ID: checkoutID, WalletID: walletID, AmountMinor: 500, Currency: "USD", Status: "pending",
	}
	payment := sqlc.Payment{
		ID: paymentID, CheckoutID: checkoutID, AmountMinor: 500, Currency: "USD",
		Status: "succeeded", VerifiedAt: pgtype.Timestamptz{Valid: true},
		WalletTransactionID: &transactionID,
	}
	credit := sqlc.WalletTransaction{
		ID: transactionID, WalletID: walletID, Direction: "credit", Reason: "topup",
		ReferenceType: "payment", ReferenceID: paymentID, AmountMinor: 500,
	}
	if err := validateCompletion(current, payment, credit); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(*sqlc.Checkout, *sqlc.Payment, *sqlc.WalletTransaction)
	}{
		{"completed checkout", func(c *sqlc.Checkout, _ *sqlc.Payment, _ *sqlc.WalletTransaction) { c.Status = "completed" }},
		{"wrong checkout", func(_ *sqlc.Checkout, p *sqlc.Payment, _ *sqlc.WalletTransaction) { p.CheckoutID = uuid.New() }},
		{"wrong wallet", func(_ *sqlc.Checkout, _ *sqlc.Payment, t *sqlc.WalletTransaction) { t.WalletID = uuid.New() }},
		{"wrong amount", func(_ *sqlc.Checkout, p *sqlc.Payment, _ *sqlc.WalletTransaction) { p.AmountMinor = 600 }},
		{"wrong currency", func(_ *sqlc.Checkout, p *sqlc.Payment, _ *sqlc.WalletTransaction) { p.Currency = "GHS" }},
		{"unverified payment", func(_ *sqlc.Checkout, p *sqlc.Payment, _ *sqlc.WalletTransaction) { p.VerifiedAt.Valid = false }},
		{"missing linked credit", func(_ *sqlc.Checkout, p *sqlc.Payment, _ *sqlc.WalletTransaction) { p.WalletTransactionID = nil }},
		{"wrong ledger entry", func(_ *sqlc.Checkout, _ *sqlc.Payment, t *sqlc.WalletTransaction) { t.ReferenceID = uuid.New() }},
		{"wrong ledger amount", func(_ *sqlc.Checkout, _ *sqlc.Payment, t *sqlc.WalletTransaction) { t.AmountMinor = 400 }},
		{"debit instead of credit", func(_ *sqlc.Checkout, _ *sqlc.Payment, t *sqlc.WalletTransaction) { t.Direction = "debit" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, p, transaction := current, payment, credit
			tc.edit(&c, &p, &transaction)
			if err := validateCompletion(c, p, transaction); err == nil {
				t.Fatal("expected checkout completion conflict")
			}
		})
	}
}
