package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
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
		return Subscription{}, err
	}

	row, err := s.repo.CreateSubscription(
		ctx,
		req.OrganizationID,
		req.PlanID,
	)
	if isUniqueViolation(err) {
		return Subscription{}, ErrSubscriptionConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrSubscriptionNotPermitted
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("create subscription: %w", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Subscription, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Subscription{}, ErrSubscriptionNotFound
	}

	row, err := s.repo.GetSubscription(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrSubscriptionNotFound
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("get subscription: %w", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) List(
	ctx context.Context,
	req ListRequest,
) ([]Subscription, error) {
	if err := normalizeListRequest(&req); err != nil {
		return nil, err
	}

	rows, err := s.repo.List(
		ctx,
		req,
	)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
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
		return Subscription{}, ErrSubscriptionNotFound
	}

	row, err := s.repo.GetCurrent(
		ctx,
		organizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrSubscriptionNotFound
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("get current subscription: %w", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Activate(
	ctx context.Context,
	req ActivateRequest,
) (Subscription, error) {
	if err := validateActivateRequest(req); err != nil {
		return Subscription{}, err
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
		return Subscription{}, ErrSubscriptionInvalidState
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("activate subscription: %w", err)
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
		return Subscription{}, ErrSubscriptionInvalidState
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("mark subscription past due: %w", err)
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
		return Subscription{}, ErrSubscriptionNotFound
	}
	if err := validateUpdateRequest(req); err != nil {
		return Subscription{}, err
	}

	row, err := s.repo.UpdateCancelAtPeriodEnd(
		ctx,
		organizationID,
		id,
		*req.CancelAtPeriodEnd,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrSubscriptionInvalidState
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("update subscription: %w", err)
	}

	return subscriptionFromRow(row), nil
}

func (s *Service) Cancel(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Subscription, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Subscription{}, ErrSubscriptionNotFound
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
		return Subscription{}, ErrSubscriptionInvalidState
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("cancel subscription: %w", err)
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
