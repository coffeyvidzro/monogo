package payments

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repo: NewRepository(db)}
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
