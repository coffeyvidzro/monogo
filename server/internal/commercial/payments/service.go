package payments

import (
	"context"
	"errors"
	"math"

	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo           *Repository
	walletRepo     *wallets.Repository
	verifiers      map[string]ProviderVerifier
	stripeClient   *stripe.Client
	paystackClient *paystack.Client
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repo: NewRepository(db), walletRepo: wallets.NewRepository(db)}
}

// ConfigureVerifiers supplies server-owned payment provider clients. This does
// not expose payment verification or wallet credits to customer API requests.
func (s *Service) ConfigureVerifiers(verifiers map[string]ProviderVerifier) {
	s.verifiers = verifiers
}

// Settle credits a checkout after the caller has independently authenticated
// and verified a successful provider charge. Payment state, wallet balance,
// immutable ledger entry, and checkout completion commit in one transaction.
// Repeating the same verified provider result returns the original result.
func (s *Service) Settle(ctx context.Context, req VerifiedSettlement) (SettlementResult, error) {
	if err := validateSettlement(&req); err != nil {
		return SettlementResult{}, err
	}
	if s == nil || !s.repo.Available() || s.walletRepo == nil {
		return SettlementResult{}, apperror.NewServiceUnavailable("payment settlement is not configured", nil)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("begin payment settlement", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repo, walletRepo := s.repo.WithTx(tx), s.walletRepo.WithTx(tx)

	payment, err := repo.Lock(ctx, req.OrganizationID, req.PaymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SettlementResult{}, apperror.NewNotFound("payment not found")
	}
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("lock payment", err)
	}
	if payment.Provider != req.Provider || payment.AmountMinor != req.AmountMinor || payment.Currency != req.Currency {
		return SettlementResult{}, apperror.NewConflict("verified settlement does not match payment attempt")
	}
	checkoutRow, err := repo.LockCheckout(ctx, payment.CheckoutID)
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("lock checkout", err)
	}

	if payment.Status == "succeeded" && payment.WalletTransactionID != nil {
		if payment.ProviderReference == nil || *payment.ProviderReference != req.ProviderReference ||
			checkoutRow.Status != "completed" || checkoutRow.CreditedTransactionID == nil ||
			*checkoutRow.CreditedTransactionID != *payment.WalletTransactionID {
			return SettlementResult{}, apperror.NewConflict("payment was settled with different provider data")
		}
		entry, getErr := walletRepo.Transaction(ctx, *payment.WalletTransactionID)
		if getErr != nil {
			return SettlementResult{}, apperror.NewInternal("read settled wallet transaction", getErr)
		}
		return SettlementResult{Payment: payment, Checkout: checkoutRow, Transaction: entry}, nil
	}
	if (payment.Status != "created" && payment.Status != "pending" && payment.Status != "succeeded") || checkoutRow.Status != "pending" {
		return SettlementResult{}, apperror.NewConflict("payment or checkout cannot be settled")
	}
	if payment.ProviderReference != nil && *payment.ProviderReference != req.ProviderReference {
		return SettlementResult{}, apperror.NewConflict("payment has a different provider reference")
	}
	wallet, err := walletRepo.LockByID(ctx, req.OrganizationID, checkoutRow.WalletID)
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("lock settlement wallet", err)
	}
	if wallet.Currency != payment.Currency || wallet.BalanceMinor > math.MaxInt64-payment.AmountMinor {
		return SettlementResult{}, apperror.NewConflict("wallet cannot accept settlement")
	}
	if payment.Status != "succeeded" {
		payment, err = repo.Succeed(ctx, payment.ID, req.ProviderReference, req.VerifiedAt)
		if err != nil {
			return SettlementResult{}, apperror.NewInternal("mark payment succeeded", err)
		}
	}
	next := wallet.BalanceMinor + payment.AmountMinor
	if _, err = walletRepo.SetBalance(ctx, wallet, req.OrganizationID, next); err != nil {
		return SettlementResult{}, apperror.NewInternal("credit prepaid wallet", err)
	}
	entry, err := walletRepo.Record(ctx, wallet.ID, wallets.Entry{OrganizationID: req.OrganizationID,
		Currency: payment.Currency, Direction: "credit", Reason: "topup", AmountMinor: payment.AmountMinor,
		ReferenceType: "payment", ReferenceID: payment.ID}, next)
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("record top-up credit", err)
	}
	payment, err = repo.LinkTransaction(ctx, payment.ID, entry.ID)
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("link payment credit", err)
	}
	checkoutRow, err = repo.CompleteCheckout(ctx, checkoutRow, payment, entry)
	if err != nil {
		return SettlementResult{}, apperror.NewInternal("complete checkout", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return SettlementResult{}, apperror.NewInternal("commit payment settlement", err)
	}
	return SettlementResult{Payment: payment, Checkout: checkoutRow, Transaction: entry}, nil
}

// Create persists an attempt. It does not submit a provider charge or credit
// the wallet; provider submission requires its own durable recovery workflow.
func (s *Service) Create(ctx context.Context, req Attempt) (sqlc.Payment, error) {
	if err := validateAttempt(&req); err != nil {
		return sqlc.Payment{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Payment{}, apperror.NewServiceUnavailable("payment persistence is not configured", nil)
	}
	row, err := s.repo.Create(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = s.repo.ByAttempt(ctx, req)
		if err == nil && (row.AmountMinor != req.AmountMinor || row.Currency != req.Currency) {
			return sqlc.Payment{}, apperror.NewConflict("payment attempt key was used for a different amount")
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Payment{}, apperror.NewConflict("checkout is not available for a payment attempt")
	}
	if err != nil {
		return sqlc.Payment{}, apperror.NewInternal("create payment attempt", err)
	}
	return row, nil
}

// CreateForCheckout creates an internal provider attempt using the checkout as
// the authoritative source for amount and currency. It does not contact the
// provider or claim that funds have been collected.
func (s *Service) CreateForCheckout(ctx context.Context, organizationID, checkoutID uuid.UUID, provider, attemptKey string) (sqlc.Payment, error) {
	if organizationID == uuid.Nil || checkoutID == uuid.Nil {
		return sqlc.Payment{}, apperror.NewBadRequest("organization and checkout are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Payment{}, apperror.NewServiceUnavailable("payment persistence is not configured", nil)
	}
	checkoutRow, err := s.repo.Checkout(ctx, organizationID, checkoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Payment{}, apperror.NewNotFound("checkout not found")
	}
	if err != nil {
		return sqlc.Payment{}, apperror.NewInternal("get payment checkout", err)
	}
	return s.Create(ctx, Attempt{OrganizationID: organizationID, CheckoutID: checkoutID,
		Provider: provider, AttemptKey: attemptKey, AmountMinor: checkoutRow.AmountMinor, Currency: checkoutRow.Currency})
}

func (s *Service) Get(ctx context.Context, organizationID, paymentID uuid.UUID) (sqlc.Payment, error) {
	if organizationID == uuid.Nil || paymentID == uuid.Nil {
		return sqlc.Payment{}, apperror.NewBadRequest("organization and payment are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Payment{}, apperror.NewServiceUnavailable("payment persistence is not configured", nil)
	}
	row, err := s.repo.Get(ctx, organizationID, paymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Payment{}, apperror.NewNotFound("payment not found")
	}
	if err != nil {
		return sqlc.Payment{}, apperror.NewInternal("get payment", err)
	}
	return row, nil
}

// RecordVerifiedEvent is internal only. Its caller MUST first authenticate
// the provider webhook. Recording an event is not proof of settlement and
// NEVER triggers a wallet credit.
func (s *Service) RecordVerifiedEvent(ctx context.Context, req Event) (sqlc.PaymentEvent, error) {
	if err := validateEvent(&req); err != nil {
		return sqlc.PaymentEvent{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.PaymentEvent{}, apperror.NewServiceUnavailable("payment persistence is not configured", nil)
	}
	row, err := s.repo.RecordEvent(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = s.repo.EventByIdentity(ctx, req)
		if err == nil && (row.PayloadSha256 != req.PayloadSHA256 || row.EventType != req.EventType) {
			return sqlc.PaymentEvent{}, apperror.NewConflict("payment event identity was used for another payload")
		}
	}
	if err != nil {
		return sqlc.PaymentEvent{}, apperror.NewInternal("record payment event", err)
	}
	return row, nil
}
