package subscriptions

import (
	"context"
	"errors"
	"fmt"
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

func (s *Service) Subscribe(
	ctx context.Context,
	req SubscribeRequest,
) (Subscription, error) {
	if err := validateSubscribeRequest(req); err != nil {
		return Subscription{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.CreateSubscription(
		ctx,
		req.OrganizationID,
		req.PlanID,
	)
	if isUniqueViolation(err) {
		return Subscription{}, apperror.NewConflict("organization already has a current subscription")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewForbidden("subscription is not permitted")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("create subscription", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Subscription, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Subscription{}, apperror.NewNotFound("subscription not found")
	}

	row, err := s.repo.GetSubscription(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewNotFound("subscription not found")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("get subscription", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) List(
	ctx context.Context,
	req ListRequest,
) ([]Subscription, error) {
	if err := normalizeListRequest(&req); err != nil {
		return nil, apperror.NewBadRequest(err.Error())
	}

	rows, err := s.repo.List(
		ctx,
		req,
	)
	if err != nil {
		return nil, apperror.NewInternal("list subscriptions", err)
	}

	result := make([]Subscription, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			subscriptionFromRow(row),
		)
	}

	return result, nil
}

func (s *Service) Current(
	ctx context.Context,
	organizationID uuid.UUID,
) (Subscription, error) {
	if organizationID == uuid.Nil {
		return Subscription{}, apperror.NewNotFound("subscription not found")
	}

	row, err := s.repo.GetCurrent(
		ctx,
		organizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewNotFound("subscription not found")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("get current subscription", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Activate(
	ctx context.Context,
	req ActivateRequest,
) (Subscription, error) {
	if err := validateActivateRequest(req); err != nil {
		return Subscription{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.Activate(
		ctx,
		sqlc.ActivateSubscriptionParams{
			CurrentPeriodStart: pgconv.TimeToTimestamptz(req.CurrentPeriodStart),
			CurrentPeriodEnd:   pgconv.TimeToTimestamptz(req.CurrentPeriodEnd),
			ID:                 req.SubscriptionID,
			OrganizationID:     req.OrganizationID,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewConflict("subscription state does not allow operation")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("activate subscription", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) MarkPastDue(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Subscription, error) {
	row, err := s.repo.MarkPastDue(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewConflict("subscription state does not allow operation")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("mark subscription past due", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req UpdateRequest,
) (Subscription, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Subscription{}, apperror.NewNotFound("subscription not found")
	}
	if err := validateUpdateRequest(req); err != nil {
		return Subscription{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.UpdateCancelAtPeriodEnd(
		ctx,
		organizationID,
		id,
		*req.CancelAtPeriodEnd,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewConflict("subscription state does not allow operation")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("update subscription", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Cancel(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Subscription, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Subscription{}, apperror.NewNotFound("subscription not found")
	}

	row, err := s.repo.Cancel(
		ctx,
		sqlc.CancelSubscriptionParams{
			CancelledAt:    pgconv.TimeToTimestamptz(s.now().UTC()),
			ID:             id,
			OrganizationID: organizationID,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, apperror.NewConflict("subscription state does not allow operation")
	}
	if err != nil {
		return Subscription{}, apperror.NewInternal("cancel subscription", err)
	}

	return subscriptionFromRow(row), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func subscriptionFromRow(row sqlc.Subscription) Subscription {
	return Subscription{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		PlanID:             row.PlanID,
		Status:             row.Status,
		Currency:           row.Currency,
		AmountMicros:       row.AmountMicros,
		Interval:           row.Interval,
		CurrentPeriodStart: pgconv.TimestamptzToTimePtr(row.CurrentPeriodStart),
		CurrentPeriodEnd:   pgconv.TimestamptzToTimePtr(row.CurrentPeriodEnd),
		CancelAtPeriodEnd:  row.CancelAtPeriodEnd,
		StartedAt:          pgconv.TimestamptzToTimePtr(row.StartedAt),
		CancelledAt:        pgconv.TimestamptzToTimePtr(row.CancelledAt),
		CreatedAt:          pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:          pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
