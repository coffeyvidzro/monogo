package checkout

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo          *Repository
	subscriptions *subscriptions.Service
	wallets       *wallets.Service
	now           func() time.Time
}

func NewService(
	repo *Repository,
	subscriptionsService *subscriptions.Service,
	walletsService *wallets.Service,
) *Service {
	return &Service{
		repo:          repo,
		subscriptions: subscriptionsService,
		wallets:       walletsService,
		now:           time.Now,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (Checkout, error) {
	if err := validateCreateRequest(&req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}

	var (
		subscriptionID *uuid.UUID
		amountMicros   int64
		currency       string
		periodStart    *time.Time
		periodEnd      *time.Time
	)

	switch req.Purpose {
	case PurposeSubscription:
		subscription, err := s.subscriptions.Get(ctx, req.OrganizationID, *req.SubscriptionID)
		if err != nil {
			return Checkout{}, err
		}
		if subscription.Status != subscriptions.StatusPending {
			return Checkout{}, apperror.NewConflict("subscription is not awaiting checkout")
		}
		if subscription.AmountMicros <= 0 {
			return Checkout{}, apperror.NewBadRequest("subscription does not require checkout")
		}

		start := s.now().UTC()
		end := start.AddDate(0, 1, 0)
		subscriptionID = &subscription.ID
		amountMicros = subscription.AmountMicros
		currency = subscription.Currency
		periodStart = &start
		periodEnd = &end

	case PurposeWalletTopup:
		wallet, err := s.wallets.Create(
			ctx,
			wallets.CreateRequest{OrganizationID: req.OrganizationID},
		)
		if err != nil {
			return Checkout{}, err
		}
		amountMicros = req.AmountMicros
		currency = wallet.Currency
	}

	row, err := s.repo.Create(
		ctx,
		req.OrganizationID,
		req.Purpose,
		subscriptionID,
		amountMicros,
		currency,
		periodStart,
		periodEnd,
	)
	if isUniqueViolation(err) {
		return Checkout{}, apperror.NewConflict("active checkout already exists")
	}
	if err != nil {
		return Checkout{}, apperror.NewInternal("create checkout", err)
	}

	return checkoutFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Checkout, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Checkout{}, apperror.NewNotFound("checkout not found")
	}

	row, err := s.repo.Get(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkout{}, apperror.NewNotFound("checkout not found")
	}
	if err != nil {
		return Checkout{}, apperror.NewInternal("get checkout", err)
	}

	return checkoutFromRow(row), nil
}

func (s *Service) Confirm(
	ctx context.Context,
	req ConfirmRequest,
) (Checkout, error) {
	if err := validateConfirmRequest(&req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.Confirm(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
		req.Provider,
		s.now().UTC(),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkout{}, apperror.NewConflict("checkout cannot be confirmed")
	}
	if err != nil {
		return Checkout{}, apperror.NewInternal("confirm checkout", err)
	}

	return checkoutFromRow(row), nil
}

func (s *Service) Continue(
	ctx context.Context,
	req ContinueRequest,
) (Checkout, error) {
	if err := validateContinueRequest(&req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.Continue(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
		req.Provider,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkout{}, apperror.NewConflict("checkout cannot continue")
	}
	if err != nil {
		return Checkout{}, apperror.NewInternal("continue checkout", err)
	}

	return checkoutFromRow(row), nil
}

func checkoutFromRow(row sqlc.Checkout) Checkout {
	return Checkout{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Purpose:        row.Purpose,
		Status:         row.Status,
		SubscriptionID: row.SubscriptionID,
		AmountMicros:   row.AmountMicros,
		Currency:       row.Currency,
		PeriodStart:    pgconv.TimestamptzToTimePtr(row.PeriodStart),
		PeriodEnd:      pgconv.TimestamptzToTimePtr(row.PeriodEnd),
		FailureCode:    row.FailureCode,
		ConfirmedAt:    pgconv.TimestamptzToTimePtr(row.ConfirmedAt),
		CompletedAt:    pgconv.TimestamptzToTimePtr(row.CompletedAt),
		FailedAt:       pgconv.TimestamptzToTimePtr(row.FailedAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
