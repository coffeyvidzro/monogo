package payments

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

func (s *Service) CreateSubscription(
	ctx context.Context,
	req CreateSubscriptionRequest,
) (Payment, error) {
	if err := validateCreateSubscriptionRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	subscription, err := s.subscriptions.Get(
		ctx,
		req.OrganizationID,
		req.SubscriptionID,
	)
	if err != nil {
		return Payment{}, err
	}
	if subscription.Status != subscriptions.StatusPending {
		return Payment{}, apperror.NewConflict("subscription is not awaiting initial payment")
	}
	if subscription.AmountMicros <= 0 {
		return Payment{}, apperror.NewBadRequest("subscription does not require payment")
	}

	periodStart := s.now().UTC()
	periodEnd := periodStart.AddDate(0, 1, 0)
	row, err := s.repo.CreateSubscription(
		ctx,
		req.OrganizationID,
		req.SubscriptionID,
		req.Provider,
		subscription.AmountMicros,
		subscription.Currency,
		periodStart,
		periodEnd,
	)
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("subscription already has a pending payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("create subscription payment", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) CreateWalletTopup(
	ctx context.Context,
	req CreateWalletTopupRequest,
) (Payment, error) {
	if err := validateCreateWalletTopupRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	wallet, err := s.wallets.Create(
		ctx,
		wallets.CreateRequest{
			OrganizationID: req.OrganizationID,
		},
	)
	if err != nil {
		return Payment{}, err
	}

	row, err := s.repo.CreateWalletTopup(
		ctx,
		req.OrganizationID,
		req.Provider,
		req.AmountMicros,
		wallet.Currency,
	)
	if err != nil {
		return Payment{}, apperror.NewInternal("create wallet top-up payment", err)
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
	req SettleRequest,
) (Payment, error) {
	if err := validateSettleRequest(&req); err != nil {
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
		return Payment{}, apperror.NewConflict("payment state does not allow settlement")
	}

	row, err := s.repo.ClaimProviderEvent(
		ctx,
		req.OrganizationID,
		req.PaymentID,
		req.ProviderEventID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment state does not allow provider event")
	}
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider event already belongs to another payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("claim payment provider event", err)
	}

	payment := paymentFromRow(row)
	switch payment.Purpose {
	case PurposeSubscription:
		if err := s.settleSubscription(ctx, payment); err != nil {
			return Payment{}, err
		}
	case PurposeWalletTopup:
		if err := s.settleWalletTopup(ctx, payment, req.OccurredAt); err != nil {
			return Payment{}, err
		}
	default:
		return Payment{}, apperror.NewInternal("settle payment", ErrConflict)
	}

	row, err = s.repo.MarkSucceeded(
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

func (s *Service) settleSubscription(
	ctx context.Context,
	payment Payment,
) error {
	if payment.SubscriptionID == nil ||
		payment.PeriodStart == nil ||
		payment.PeriodEnd == nil {
		return apperror.NewInternal("settle subscription payment", ErrConflict)
	}

	subscription, err := s.subscriptions.Get(
		ctx,
		payment.OrganizationID,
		*payment.SubscriptionID,
	)
	if err != nil {
		return err
	}
	if subscription.Status == subscriptions.StatusActive &&
		sameTime(subscription.CurrentPeriodStart, *payment.PeriodStart) &&
		sameTime(subscription.CurrentPeriodEnd, *payment.PeriodEnd) {
		return nil
	}

	_, err = s.subscriptions.Activate(
		ctx,
		subscriptions.ActivateRequest{
			OrganizationID:     payment.OrganizationID,
			SubscriptionID:     *payment.SubscriptionID,
			CurrentPeriodStart: *payment.PeriodStart,
			CurrentPeriodEnd:   *payment.PeriodEnd,
		},
	)
	return err
}

func (s *Service) settleWalletTopup(
	ctx context.Context,
	payment Payment,
	occurredAt time.Time,
) error {
	referenceType := "payment"
	referenceID := payment.ID
	_, err := s.wallets.Credit(
		ctx,
		wallets.MovementRequest{
			OrganizationID: payment.OrganizationID,
			OperationID:    payment.ID,
			AmountMicros:   payment.AmountMicros,
			Reason:         "wallet_topup",
			ReferenceType:  &referenceType,
			ReferenceID:    &referenceID,
			OccurredAt:     occurredAt,
		},
	)
	return err
}

func sameTime(value *time.Time, expected time.Time) bool {
	return value != nil && value.Equal(expected)
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
