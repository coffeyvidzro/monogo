package payments

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (Payment, error) {
	if err := validateCreateRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	var (
		row sqlc.Payment
		err error
	)

	switch req.Purpose {
	case PurposeSubscription:
		row, err = s.repo.CreateSubscription(
			ctx,
			req.OrganizationID,
			*req.SubscriptionID,
			req.Provider,
			req.AmountMicros,
			req.Currency,
			*req.PeriodStart,
			*req.PeriodEnd,
		)
	case PurposeWalletTopup:
		row, err = s.repo.CreateWalletTopup(
			ctx,
			req.OrganizationID,
			req.Provider,
			req.AmountMicros,
			req.Currency,
		)
	}
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("payment conflicts with existing payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("create payment", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Payment, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Payment{}, apperror.NewNotFound("payment not found")
	}

	row, err := s.repo.Get(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewNotFound("payment not found")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("get payment", err)
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
		return Payment{}, apperror.NewConflict("payment state does not allow provider reference")
	}
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider reference already belongs to another payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("attach payment provider reference", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) Succeed(
	ctx context.Context,
	req SucceedRequest,
) (Payment, error) {
	if err := validateSucceedRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}
	if req.OccurredAt.IsZero() {
		req.OccurredAt = s.now().UTC()
	}

	current, err := s.Get(ctx, req.OrganizationID, req.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if current.Status == StatusSucceeded {
		if current.ProviderEventID != nil && *current.ProviderEventID == req.ProviderEventID {
			return current, nil
		}
		return Payment{}, apperror.NewConflict("payment already succeeded with another provider event")
	}
	if current.Status != StatusPending {
		return Payment{}, apperror.NewConflict("payment state does not allow success")
	}

	if _, err := s.repo.ClaimProviderEvent(
		ctx,
		req.OrganizationID,
		req.PaymentID,
		req.ProviderEventID,
	); errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment state does not allow provider event")
	} else if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider event already belongs to another payment")
	} else if err != nil {
		return Payment{}, apperror.NewInternal("claim payment provider event", err)
	}

	row, err := s.repo.MarkSucceeded(
		ctx,
		req.OrganizationID,
		req.PaymentID,
		req.ProviderEventID,
		req.OccurredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment state does not allow success")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("mark payment succeeded", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) Fail(
	ctx context.Context,
	req FailRequest,
) (Payment, error) {
	if err := validateFailRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}
	if req.OccurredAt.IsZero() {
		req.OccurredAt = s.now().UTC()
	}

	current, err := s.Get(ctx, req.OrganizationID, req.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if current.Status == StatusFailed {
		if current.ProviderEventID != nil &&
			*current.ProviderEventID == req.ProviderEventID &&
			current.FailureCode != nil &&
			*current.FailureCode == req.FailureCode {
			return current, nil
		}
		return Payment{}, apperror.NewConflict("payment already failed with different provider data")
	}
	if current.Status != StatusPending {
		return Payment{}, apperror.NewConflict("payment state does not allow failure")
	}

	if _, err := s.repo.ClaimProviderEvent(
		ctx,
		req.OrganizationID,
		req.PaymentID,
		req.ProviderEventID,
	); errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment state does not allow provider event")
	} else if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider event already belongs to another payment")
	} else if err != nil {
		return Payment{}, apperror.NewInternal("claim payment provider event", err)
	}

	row, err := s.repo.MarkFailed(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment state does not allow failure")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("mark payment failed", err)
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
		OrganizationID:    row.OrganizationID,
		Purpose:           row.Purpose,
		Provider:          row.Provider,
		SubscriptionID:    row.SubscriptionID,
		AmountMicros:      row.AmountMicros,
		Currency:          row.Currency,
		Status:            row.Status,
		ProviderReference: row.ProviderReference,
		ProviderEventID:   row.ProviderEventID,
		PeriodStart:       pgconv.TimestamptzToTimePtr(row.PeriodStart),
		PeriodEnd:         pgconv.TimestamptzToTimePtr(row.PeriodEnd),
		FailureCode:       row.FailureCode,
		CompletedAt:       pgconv.TimestamptzToTimePtr(row.CompletedAt),
		CreatedAt:         pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:         pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
