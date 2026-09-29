package checkout

import (
	"context"
	"errors"
	"strings"
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

const checkoutTTL = 30 * time.Minute

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

	now := s.now().UTC()
	var (
		subscriptionID *uuid.UUID
		amountMicros   int64
		currency       string
	)

	switch req.Purpose {
	case PurposeSubscription:
		subscription, err := s.subscriptions.Get(
			ctx,
			req.OrganizationID,
			*req.SubscriptionID,
		)
		if err != nil {
			return Checkout{}, err
		}
		if subscription.Status != subscriptions.StatusPending {
			return Checkout{}, apperror.NewConflict("subscription is not awaiting checkout")
		}
		if subscription.AmountMicros <= 0 {
			return Checkout{}, apperror.NewBadRequest("subscription does not require checkout")
		}

		subscriptionID = &subscription.ID
		amountMicros = subscription.AmountMicros
		currency = subscription.Currency

	case PurposeWalletTopup:
		wallet, err := s.wallets.Create(
			ctx,
			wallets.CreateRequest{
				OrganizationID: req.OrganizationID,
			},
		)
		if err != nil {
			return Checkout{}, err
		}

		amountMicros = req.AmountMicros
		currency = wallet.Currency
	}

	reference := newReference()
	row, err := s.repo.Create(
		ctx,
		req.OrganizationID,
		req.Purpose,
		subscriptionID,
		reference,
		amountMicros,
		currency,
		now.Add(checkoutTTL),
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

	row, err := s.repo.Confirm(ctx, req, s.now().UTC())
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
	if err := validateContinueRequest(req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.GetForContinuation(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
		s.now().UTC(),
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
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Purpose:         row.Purpose,
		SubscriptionID:  row.SubscriptionID,
		Reference:       row.Reference,
		AmountMicros:    row.AmountMicros,
		Currency:        row.Currency,
		Status:          row.Status,
		Provider:        row.Provider,
		PaymentMethod:   row.PaymentMethod,
		NextAction:      row.NextAction,
		ProviderMessage: row.ProviderMessage,
		FailureCode:     row.FailureCode,
		ExpiresAt:       pgconv.TimestamptzToTime(row.ExpiresAt),
		CompletedAt:     pgconv.TimestamptzToTimePtr(row.CompletedAt),
		CreatedAt:       pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:       pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

func newReference() string {
	return "co_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Service) Complete(
	ctx context.Context,
	organizationID uuid.UUID,
	checkoutID uuid.UUID,
	completedAt time.Time,
) (Checkout, error) {
	current, err := s.Get(ctx, organizationID, checkoutID)
	if err != nil {
		return Checkout{}, err
	}
	if current.Status == StatusSucceeded {
		return current, nil
	}
	if current.Status != StatusProcessing {
		return Checkout{}, apperror.NewConflict("checkout cannot complete")
	}
	if completedAt.IsZero() {
		completedAt = s.now().UTC()
	}

	switch current.Purpose {
	case PurposeSubscription:
		if err := s.completeSubscription(ctx, current, completedAt); err != nil {
			return Checkout{}, err
		}
	case PurposeWalletTopup:
		if err := s.completeWalletTopup(ctx, current, completedAt); err != nil {
			return Checkout{}, err
		}
	default:
		return Checkout{}, apperror.NewInternal("complete checkout", ErrInvalidInput)
	}

	row, err := s.repo.Complete(ctx, organizationID, checkoutID, completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkout{}, apperror.NewConflict("checkout cannot complete")
	}
	if err != nil {
		return Checkout{}, apperror.NewInternal("complete checkout", err)
	}

	return checkoutFromRow(row), nil
}

func (s *Service) Fail(
	ctx context.Context,
	organizationID uuid.UUID,
	checkoutID uuid.UUID,
	failureCode string,
	providerMessage *string,
	completedAt time.Time,
) (Checkout, error) {
	current, err := s.Get(ctx, organizationID, checkoutID)
	if err != nil {
		return Checkout{}, err
	}
	if current.Status == StatusFailed {
		return current, nil
	}
	if current.Status != StatusProcessing {
		return Checkout{}, apperror.NewConflict("checkout cannot fail")
	}
	failureCode = strings.TrimSpace(failureCode)
	if failureCode == "" {
		return Checkout{}, apperror.NewBadRequest("failure code is required")
	}
	if completedAt.IsZero() {
		completedAt = s.now().UTC()
	}

	row, err := s.repo.Fail(
		ctx,
		organizationID,
		checkoutID,
		failureCode,
		providerMessage,
		completedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checkout{}, apperror.NewConflict("checkout cannot fail")
	}
	if err != nil {
		return Checkout{}, apperror.NewInternal("fail checkout", err)
	}

	return checkoutFromRow(row), nil
}

func (s *Service) completeSubscription(
	ctx context.Context,
	checkout Checkout,
	completedAt time.Time,
) error {
	if checkout.SubscriptionID == nil {
		return apperror.NewInternal("complete subscription checkout", ErrInvalidInput)
	}

	subscription, err := s.subscriptions.Get(
		ctx,
		checkout.OrganizationID,
		*checkout.SubscriptionID,
	)
	if err != nil {
		return err
	}

	if subscription.Status == subscriptions.StatusActive {
		return nil
	}

	periodStart := completedAt.UTC()
	periodEnd := periodStart.AddDate(0, 1, 0)

	_, err = s.subscriptions.Activate(
		ctx,
		subscriptions.ActivateRequest{
			OrganizationID:     checkout.OrganizationID,
			SubscriptionID:     *checkout.SubscriptionID,
			CurrentPeriodStart: periodStart,
			CurrentPeriodEnd:   periodEnd,
		},
	)
	return err
}

func (s *Service) completeWalletTopup(
	ctx context.Context,
	checkout Checkout,
	completedAt time.Time,
) error {
	referenceType := "checkout"
	referenceID := checkout.ID

	_, err := s.wallets.Credit(
		ctx,
		wallets.MovementRequest{
			OrganizationID: checkout.OrganizationID,
			OperationID:    checkout.ID,
			AmountMicros:   checkout.AmountMicros,
			Reason:         "wallet_topup",
			ReferenceType:  &referenceType,
			ReferenceID:    &referenceID,
			OccurredAt:     completedAt,
		},
	)
	return err
}
