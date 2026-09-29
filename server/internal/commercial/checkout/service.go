package checkout

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type Service struct {
	payments      *payments.Service
	subscriptions *subscriptions.Service
	wallets       *wallets.Service
	now           func() time.Time
}

func NewService(
	paymentsService *payments.Service,
	subscriptionsService *subscriptions.Service,
	walletsService *wallets.Service,
) *Service {
	return &Service{
		payments:      paymentsService,
		subscriptions: subscriptionsService,
		wallets:       walletsService,
		now:           time.Now,
	}
}

func (s *Service) CreateSubscription(
	ctx context.Context,
	req CreateSubscriptionRequest,
) (Checkout, error) {
	if err := validateCreateSubscriptionRequest(req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}

	subscription, err := s.subscriptions.Get(
		ctx,
		req.OrganizationID,
		req.SubscriptionID,
	)
	if err != nil {
		return Checkout{}, err
	}
	if subscription.Status != subscriptions.StatusPending {
		return Checkout{}, apperror.NewConflict("subscription is not awaiting initial payment")
	}
	if subscription.AmountMicros <= 0 {
		return Checkout{}, apperror.NewBadRequest("subscription does not require payment")
	}

	periodStart := s.now().UTC()
	periodEnd := periodStart.AddDate(0, 1, 0)
	subscriptionID := subscription.ID
	payment, err := s.payments.Create(
		ctx,
		payments.CreateRequest{
			OrganizationID: req.OrganizationID,
			Purpose:        payments.PurposeSubscription,
			Provider:       req.Provider,
			SubscriptionID: &subscriptionID,
			AmountMicros:   subscription.AmountMicros,
			Currency:       subscription.Currency,
			PeriodStart:    &periodStart,
			PeriodEnd:      &periodEnd,
		},
	)
	if err != nil {
		return Checkout{}, err
	}

	return Checkout{Payment: payment}, nil
}

func (s *Service) CreateWalletTopup(
	ctx context.Context,
	req CreateWalletTopupRequest,
) (Checkout, error) {
	if err := validateCreateWalletTopupRequest(req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}

	wallet, err := s.wallets.Create(
		ctx,
		wallets.CreateRequest{
			OrganizationID: req.OrganizationID,
		},
	)
	if err != nil {
		return Checkout{}, err
	}

	payment, err := s.payments.Create(
		ctx,
		payments.CreateRequest{
			OrganizationID: req.OrganizationID,
			Purpose:        payments.PurposeWalletTopup,
			Provider:       req.Provider,
			AmountMicros:   req.AmountMicros,
			Currency:       wallet.Currency,
		},
	)
	if err != nil {
		return Checkout{}, err
	}

	return Checkout{Payment: payment}, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	paymentID uuid.UUID,
) (Checkout, error) {
	payment, err := s.payments.Get(ctx, organizationID, paymentID)
	if err != nil {
		return Checkout{}, err
	}

	return Checkout{Payment: payment}, nil
}

func (s *Service) Complete(
	ctx context.Context,
	req CompleteRequest,
) (Checkout, error) {
	if err := validateCompleteRequest(req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}
	if req.OccurredAt.IsZero() {
		req.OccurredAt = s.now().UTC()
	}

	payment, err := s.payments.Succeed(
		ctx,
		payments.SucceedRequest{
			OrganizationID:  req.OrganizationID,
			PaymentID:       req.PaymentID,
			ProviderEventID: req.ProviderEventID,
			OccurredAt:      req.OccurredAt,
		},
	)
	if err != nil {
		return Checkout{}, err
	}

	if err := s.applyCompletion(ctx, payment, req.OccurredAt); err != nil {
		return Checkout{}, err
	}

	return Checkout{Payment: payment}, nil
}

func (s *Service) Fail(
	ctx context.Context,
	req FailRequest,
) (Checkout, error) {
	if err := validateFailRequest(req); err != nil {
		return Checkout{}, apperror.NewBadRequest(err.Error())
	}
	if req.OccurredAt.IsZero() {
		req.OccurredAt = s.now().UTC()
	}

	payment, err := s.payments.Fail(
		ctx,
		payments.FailRequest{
			OrganizationID:  req.OrganizationID,
			PaymentID:       req.PaymentID,
			ProviderEventID: req.ProviderEventID,
			FailureCode:     req.FailureCode,
			OccurredAt:      req.OccurredAt,
		},
	)
	if err != nil {
		return Checkout{}, err
	}

	return Checkout{Payment: payment}, nil
}

func (s *Service) applyCompletion(
	ctx context.Context,
	payment payments.Payment,
	occurredAt time.Time,
) error {
	switch payment.Purpose {
	case payments.PurposeSubscription:
		return s.completeSubscription(ctx, payment)
	case payments.PurposeWalletTopup:
		return s.completeWalletTopup(ctx, payment, occurredAt)
	default:
		return apperror.NewInternal("complete checkout", payments.ErrConflict)
	}
}

func (s *Service) completeSubscription(
	ctx context.Context,
	payment payments.Payment,
) error {
	if payment.SubscriptionID == nil ||
		payment.PeriodStart == nil ||
		payment.PeriodEnd == nil {
		return apperror.NewInternal("complete subscription checkout", payments.ErrConflict)
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

func (s *Service) completeWalletTopup(
	ctx context.Context,
	payment payments.Payment,
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
