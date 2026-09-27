package subscriptions

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	if repo == nil {
		panic("billing subscriptions: repository is required")
	}

	return &Service{
		repo: repo,
	}
}

func (s *Service) CreatePlan(
	ctx context.Context,
	req CreatePlanRequest,
) (sqlc.SubscriptionPlan, error) {
	if err := normalizePlan(&req); err != nil {
		return sqlc.SubscriptionPlan{}, err
	}

	plan, err := s.repo.CreatePlan(
		ctx,
		req,
	)
	if err == nil {
		return plan, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" {
		return sqlc.SubscriptionPlan{}, apperror.NewConflict(
			"subscription plan code already exists",
		)
	}

	return sqlc.SubscriptionPlan{}, apperror.NewInternal(
		"create subscription plan",
		err,
	)
}

func (s *Service) GetPlanByCode(
	ctx context.Context,
	code string,
) (sqlc.SubscriptionPlan, error) {
	code, err := normalizePlanCode(code)
	if err != nil {
		return sqlc.SubscriptionPlan{}, err
	}

	plan, err := s.repo.GetPlanByCode(
		ctx,
		code,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.SubscriptionPlan{}, apperror.NewNotFound(
			"subscription plan not found",
		)
	}
	if err != nil {
		return sqlc.SubscriptionPlan{}, apperror.NewInternal(
			"get subscription plan",
			err,
		)
	}

	return plan, nil
}

func (s *Service) ListPlans(
	ctx context.Context,
) ([]sqlc.SubscriptionPlan, error) {
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, apperror.NewInternal(
			"list subscription plans",
			err,
		)
	}

	return plans, nil
}

func (s *Service) ArchivePlan(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	if id == uuid.Nil {
		return sqlc.SubscriptionPlan{}, apperror.NewBadRequest(
			"plan id is required",
		)
	}

	plan, err := s.repo.ArchivePlan(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.SubscriptionPlan{}, apperror.NewNotFound(
			"subscription plan not found",
		)
	}
	if err != nil {
		return sqlc.SubscriptionPlan{}, apperror.NewInternal(
			"archive subscription plan",
			err,
		)
	}

	return plan, nil
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (sqlc.Subscription, error) {
	if err := validateCreate(req); err != nil {
		return sqlc.Subscription{}, err
	}

	value, err := s.repo.Create(
		ctx,
		req,
	)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Subscription{}, apperror.NewInternal(
			"create subscription",
			err,
		)
	}

	current, currentErr := s.repo.Current(
		ctx,
		req.OrganizationID,
	)
	if currentErr == nil {
		if current.PlanID != req.PlanID {
			return sqlc.Subscription{}, apperror.NewConflict(
				"organization already has a current subscription",
			)
		}

		return current, nil
	}

	if errors.Is(currentErr, pgx.ErrNoRows) {
		return sqlc.Subscription{}, apperror.NewNotFound(
			"organization or active plan not found",
		)
	}

	return sqlc.Subscription{}, apperror.NewInternal(
		"get current subscription",
		currentErr,
	)
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	if err := validateSubscriptionIdentity(
		organizationID,
		id,
	); err != nil {
		return sqlc.Subscription{}, err
	}

	value, err := s.repo.Get(
		ctx,
		organizationID,
		id,
	)

	return value, subscriptionReadError(err)
}

func (s *Service) Current(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Subscription, error) {
	if organizationID == uuid.Nil {
		return sqlc.Subscription{}, apperror.NewBadRequest(
			"organization context required",
		)
	}

	value, err := s.repo.Current(
		ctx,
		organizationID,
	)

	return value, subscriptionReadError(err)
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Subscription, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest(
			"organization context required",
		)
	}

	values, err := s.repo.List(
		ctx,
		organizationID,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list subscriptions",
			err,
		)
	}

	return values, nil
}

func (s *Service) Activate(
	ctx context.Context,
	req PeriodRequest,
) (sqlc.Subscription, error) {
	if err := validatePeriod(req); err != nil {
		return sqlc.Subscription{}, err
	}

	value, err := s.repo.Activate(
		ctx,
		req,
	)

	return value, subscriptionStateError(
		err,
		"activate subscription",
	)
}

func (s *Service) MarkPastDue(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	if err := validateSubscriptionIdentity(
		organizationID,
		id,
	); err != nil {
		return sqlc.Subscription{}, err
	}

	value, err := s.repo.MarkPastDue(
		ctx,
		organizationID,
		id,
	)

	return value, subscriptionStateError(
		err,
		"mark subscription past due",
	)
}

func (s *Service) SetCancelAtPeriodEnd(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	value bool,
) (sqlc.Subscription, error) {
	if err := validateSubscriptionIdentity(
		organizationID,
		id,
	); err != nil {
		return sqlc.Subscription{}, err
	}

	subscription, err := s.repo.SetCancelAtPeriodEnd(
		ctx,
		organizationID,
		id,
		value,
	)

	return subscription, subscriptionStateError(
		err,
		"update subscription cancellation",
	)
}

func (s *Service) Renew(
	ctx context.Context,
	req PeriodRequest,
) (sqlc.Subscription, error) {
	if err := validatePeriod(req); err != nil {
		return sqlc.Subscription{}, err
	}

	value, err := s.repo.Renew(
		ctx,
		req,
	)

	return value, subscriptionStateError(
		err,
		"renew subscription",
	)
}

func (s *Service) Cancel(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	if err := validateSubscriptionIdentity(
		organizationID,
		id,
	); err != nil {
		return sqlc.Subscription{}, err
	}

	value, err := s.repo.Cancel(
		ctx,
		organizationID,
		id,
	)

	return value, subscriptionStateError(
		err,
		"cancel subscription",
	)
}

func subscriptionReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(
			"subscription not found",
		)
	}
	if err != nil {
		return apperror.NewInternal(
			"get subscription",
			err,
		)
	}

	return nil
}

func subscriptionStateError(
	err error,
	action string,
) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewConflict(
			action + " is not valid in the current subscription state",
		)
	}
	if err != nil {
		return apperror.NewInternal(
			action,
			err,
		)
	}

	return nil
}
