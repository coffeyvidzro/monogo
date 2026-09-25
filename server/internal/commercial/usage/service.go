package usage

import (
	"context"
	"errors"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo       *Repository
	walletRepo *wallets.Repository
}

func NewService(repo *Repository, walletRepositories ...*wallets.Repository) *Service {
	var walletRepo *wallets.Repository
	if len(walletRepositories) > 0 {
		walletRepo = walletRepositories[0]
	}
	return &Service{repo: repo, walletRepo: walletRepo}
}

// Charge records an immutable observation and atomically authorizes and debits
// its trusted rated amount. The immutable wallet transaction uses the usage
// event as its business reference, so no parallel billing record is required.
func (s *Service) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if err := validateRecord(&req.Usage); err != nil {
		return ChargeResult{}, err
	}
	if req.AmountMinor <= 0 || !validCurrency(req.Currency) {
		return ChargeResult{}, apperror.NewBadRequest("positive rated amount and currency are required")
	}
	if s == nil || s.repo == nil || s.repo.db == nil || s.walletRepo == nil {
		return ChargeResult{}, apperror.NewServiceUnavailable("usage charging is not configured", nil)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return ChargeResult{}, apperror.NewInternal("begin usage charge", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repo, walletRepo := s.repo.WithTx(tx), s.walletRepo.WithTx(tx)
	event, err := record(ctx, repo, req.Usage)
	if err != nil {
		return ChargeResult{}, err
	}

	wallet, err := walletRepo.Lock(ctx, req.Usage.OrganizationID, req.Currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChargeResult{}, apperror.NewPaymentRequired("prepaid wallet or funds unavailable")
	}
	if err != nil {
		return ChargeResult{}, apperror.NewInternal("lock usage wallet", err)
	}
	existing, err := walletRepo.FindTransaction(ctx, wallet.ID, "usage_event", event.ID)
	if err == nil {
		if existing.Direction != "debit" || existing.Reason != "usage" || existing.AmountMinor != req.AmountMinor {
			return ChargeResult{}, apperror.NewConflict("usage observation was charged with a different rating")
		}
		return ChargeResult{UsageEvent: event, Transaction: existing}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ChargeResult{}, apperror.NewInternal("read usage charge", err)
	}
	if wallet.BalanceMinor < req.AmountMinor {
		return ChargeResult{}, apperror.NewPaymentRequired("insufficient prepaid balance")
	}
	next := wallet.BalanceMinor - req.AmountMinor
	if _, err = walletRepo.SetBalance(ctx, wallet, req.Usage.OrganizationID, next); err != nil {
		return ChargeResult{}, apperror.NewInternal("debit usage wallet", err)
	}
	entry, err := walletRepo.Record(ctx, wallet.ID, wallets.Entry{OrganizationID: req.Usage.OrganizationID,
		Currency: req.Currency, Direction: "debit", Reason: "usage", AmountMinor: req.AmountMinor,
		ReferenceType: "usage_event", ReferenceID: event.ID}, next)
	if err != nil {
		return ChargeResult{}, apperror.NewInternal("record usage debit", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ChargeResult{}, apperror.NewInternal("commit usage charge", err)
	}
	return ChargeResult{UsageEvent: event, Transaction: entry}, nil
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, ch := range value {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

// Record is internal-only. Usage recording neither charges a wallet nor
// establishes a billable amount; billing authorization is a separate workflow.
func (s *Service) Record(ctx context.Context, req RecordRequest) (sqlc.UsageEvent, error) {
	if err := validateRecord(&req); err != nil {
		return sqlc.UsageEvent{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.UsageEvent{}, apperror.NewServiceUnavailable("usage persistence is not configured", nil)
	}
	return record(ctx, s.repo, req)
}

func record(ctx context.Context, repo *Repository, req RecordRequest) (sqlc.UsageEvent, error) {
	event, err := repo.Record(ctx, req)
	if err == nil {
		return event, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewInternal("record usage observation", err)
	}
	// ON CONFLICT DO NOTHING returns no row for an existing key. Verify
	// every material field, not just the idempotency key, before replaying.
	existing, err := repo.ByKey(ctx, req.OrganizationID, req.IdempotencyKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewConflict("meter unavailable for usage recording")
	}
	if err != nil {
		return sqlc.UsageEvent{}, apperror.NewInternal("resolve usage observation", err)
	}
	matched, err := repo.MatchingByKey(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewConflict("usage idempotency key already used for another observation")
	}
	if err != nil {
		return sqlc.UsageEvent{}, apperror.NewInternal("verify usage observation replay", err)
	}
	if matched.ID != existing.ID {
		return sqlc.UsageEvent{}, apperror.NewConflict("usage observation replay mismatch")
	}
	return existing, nil
}

func (s *Service) Get(ctx context.Context, organizationID, eventID uuid.UUID) (sqlc.UsageEvent, error) {
	if organizationID == uuid.Nil || eventID == uuid.Nil {
		return sqlc.UsageEvent{}, apperror.NewBadRequest("organization and usage event are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.UsageEvent{}, apperror.NewServiceUnavailable("usage persistence is not configured", nil)
	}
	event, err := s.repo.Get(ctx, organizationID, eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewNotFound("usage event not found")
	}
	if err != nil {
		return sqlc.UsageEvent{}, apperror.NewInternal("get usage event", err)
	}
	return event, nil
}
