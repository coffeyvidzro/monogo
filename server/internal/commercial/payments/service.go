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

func (s *Service) AttachProviderReference(
	ctx context.Context,
	req AttachProviderReferenceRequest,
) (Payment, error) {
	if err := validateAttachProviderReferenceRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.AttachProviderReference(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment attempt does not allow provider reference")
	}
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider reference already belongs to another payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("attach provider reference", err)
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
		ProviderReference: row.ProviderReference,
		FailureCode:       row.FailureCode,
		CompletedAt:       pgconv.TimestamptzToTimePtr(row.CompletedAt),
		CreatedAt:         pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:         pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
