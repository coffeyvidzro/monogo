package checkout

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/jackc/pgx/v5"
)

// CompleteWithCredit finalizes an already-verified checkout inside the same
// transaction that credited the wallet, inserted its immutable ledger entry,
// and linked that entry to the successful payment. The caller must hold the
// checkout row lock and must commit or roll back the surrounding transaction.
func CompleteWithCredit(
	ctx context.Context,
	queries *sqlc.Queries,
	current sqlc.Checkout,
	payment sqlc.Payment,
	credit sqlc.WalletTransaction,
) (sqlc.Checkout, error) {
	if queries == nil {
		return sqlc.Checkout{}, apperror.NewServiceUnavailable("checkout completion is not configured", nil)
	}
	if err := validateCompletion(current, payment, credit); err != nil {
		return sqlc.Checkout{}, err
	}
	completed, err := queries.CompleteWalletCheckout(ctx, sqlc.CompleteWalletCheckoutParams{
		ID:                    current.ID,
		CreditedTransactionID: credit.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Checkout{}, apperror.NewConflict("checkout is no longer pending")
	}
	if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("complete verified checkout", err)
	}
	return completed, nil
}

// validateCompletion prevents a credit from a different payment or wallet
// from being associated with this checkout. Payment verification itself must
// already have been performed by the trusted payment settlement boundary.
func validateCompletion(
	current sqlc.Checkout,
	payment sqlc.Payment,
	credit sqlc.WalletTransaction,
) error {
	if current.Status != "pending" ||
		current.ID != payment.CheckoutID ||
		current.WalletID != credit.WalletID ||
		current.AmountMinor != payment.AmountMinor ||
		current.Currency != payment.Currency ||
		payment.Status != "succeeded" ||
		payment.VerifiedAt.Valid == false ||
		payment.WalletTransactionID == nil ||
		*payment.WalletTransactionID != credit.ID ||
		credit.Direction != "credit" ||
		credit.Reason != "topup" ||
		credit.ReferenceType != "payment" ||
		credit.ReferenceID != payment.ID ||
		credit.AmountMinor != payment.AmountMinor {
		return apperror.NewConflict("checkout, verified payment, and wallet credit do not match")
	}
	return nil
}
