package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func (s *Service) CreatePlan(
	ctx context.Context,
	req CreatePlanRequest,
) (Plan, error) {
	if err := normalizeCreatePlanRequest(&req); err != nil {
		return Plan{}, err
	}

	row, err := s.repo.CreatePlan(
		ctx,
		sqlc.CreateSubscriptionPlanParams{
			Code:         req.Code,
			Name:         req.Name,
			Currency:     req.Currency,
			AmountMicros: req.AmountMicros,
		},
	)
	if isUniqueViolation(err) {
		return Plan{}, ErrPlanConflict
	}
	if err != nil {
		return Plan{}, fmt.Errorf("create subscription plan: %w", err)
	}

	return planFromRow(row), nil
}

func (s *Service) GetPlan(
	ctx context.Context,
	id uuid.UUID,
) (Plan, error) {
	if id == uuid.Nil {
		return Plan{}, ErrPlanNotFound
	}

	row, err := s.repo.GetPlanByID(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("get subscription plan: %w", err)
	}

	return planFromRow(row), nil
}

func (s *Service) GetPlanByCode(
	ctx context.Context,
	code string,
) (Plan, error) {
	req := CreatePlanRequest{
		Code:     code,
		Name:     "placeholder",
		Currency: "USD",
	}
	req.Code = normalizePlanCode(req.Code)
	if !planCodePattern.MatchString(req.Code) {
		return Plan{}, ErrPlanNotFound
	}

	row, err := s.repo.GetPlanByCode(
		ctx,
		req.Code,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("get subscription plan by code: %w", err)
	}

	return planFromRow(row), nil
}

func (s *Service) ListPlans(
	ctx context.Context,
) ([]Plan, error) {
	rows, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subscription plans: %w", err)
	}

	result := make([]Plan, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			planFromRow(row),
		)
	}

	return result, nil
}

func (s *Service) ArchivePlan(
	ctx context.Context,
	id uuid.UUID,
) (Plan, error) {
	if id == uuid.Nil {
		return Plan{}, ErrPlanNotFound
	}

	row, err := s.repo.ArchivePlan(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("archive subscription plan: %w", err)
	}

	return planFromRow(row), nil
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

func (s *Service) CancelAtPeriodEnd(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Subscription, error) {
	row, err := s.repo.SetCancelAtPeriodEnd(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrSubscriptionInvalidState
	}
	if err != nil {
		return Subscription{}, fmt.Errorf("schedule subscription cancellation: %w", err)
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

func normalizePlanCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func planFromRow(row sqlc.SubscriptionPlan) Plan {
	return Plan{
		ID:           row.ID,
		Code:         row.Code,
		Name:         row.Name,
		Currency:     row.Currency,
		Interval:     row.Interval,
		AmountMicros: row.AmountMicros,
		Status:       row.Status,
		CreatedAt:    pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:    pgconv.TimestamptzToTime(row.UpdatedAt),
	}
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
