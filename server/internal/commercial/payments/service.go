package payments

import (
	"context"
	"errors"
	"math"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateAttempt(
	ctx context.Context,
	req CreateAttemptRequest,
) (Payment, error) {
	if err := validateCreateAttemptRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	existing, err := s.repo.GetActiveByCheckout(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
	)
	if err == nil {
		return paymentFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewInternal("get active payment attempt", err)
	}

	nextAttempt, err := s.repo.NextAttempt(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
	)
	if err != nil {
		return Payment{}, apperror.NewInternal("get next payment attempt", err)
	}
	if nextAttempt <= 0 || nextAttempt > math.MaxInt32 {
		return Payment{}, apperror.NewInternal("create payment attempt", ErrConflict)
	}

	row, err := s.repo.CreateAttempt(
		ctx,
		req,
		int32(nextAttempt),
	)
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("payment attempt conflict")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("create payment attempt", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) AttachProviderPaymentID(
	ctx context.Context,
	req AttachProviderPaymentIDRequest,
) (Payment, error) {
	if err := validateAttachProviderPaymentIDRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.AttachProviderPaymentID(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment attempt does not allow provider payment id")
	}
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider payment id already belongs to another payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("attach provider payment id", err)
	}

	return paymentFromRow(row), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func paymentFromRow(row sqlc.Payment) Payment {
	return Payment{
		ID:                row.ID,
		CheckoutID:        row.CheckoutID,
		OrganizationID:    row.OrganizationID,
		Provider:          row.Provider,
		Attempt:           row.Attempt,
		AmountMicros:      row.AmountMicros,
		Currency:          row.Currency,
		Status:            row.Status,
		ProviderPaymentID: row.ProviderPaymentID,
		FailureCode:       row.FailureCode,
		PaidAt:            pgconv.TimestamptzToTimePtr(row.PaidAt),
		CreatedAt:         pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:         pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
