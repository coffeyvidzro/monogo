package payments

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type InitiationResult struct {
	Payment        sqlc.Payment
	ClientSecret   string
	DisplayText    string
	ProviderStatus string
}

func (s *Service) ConfigureInitiation(stripeClient *stripe.Client, paystackClient *paystack.Client) {
	s.stripeClient = stripeClient
	s.paystackClient = paystackClient
}

// Initiate submits only the first successful database claimant. Never retry an
// uncertain provider request: a network failure may follow a successful charge.
// The durable pending payment must first be reconciled with its provider.
func (s *Service) Initiate(
	ctx context.Context,
	organizationID, checkoutID uuid.UUID,
	attemptKey string,
	input CreateInput,
) (InitiationResult, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	if organizationID == uuid.Nil || checkoutID == uuid.Nil ||
		strings.TrimSpace(attemptKey) == "" {
		return InitiationResult{}, apperror.NewBadRequest("organization, checkout and idempotency key are required")
	}
	if s == nil || !s.repo.Available() {
		return InitiationResult{}, apperror.NewServiceUnavailable("payment persistence is not configured", nil)
	}
	switch input.Provider {
	case "stripe":
		if s.stripeClient == nil {
			return InitiationResult{}, apperror.NewServiceUnavailable("stripe is not configured", nil)
		}
	case "paystack":
		if s.paystackClient == nil {
			return InitiationResult{}, apperror.NewServiceUnavailable("paystack is not configured", nil)
		}
		if err := (paystack.ChargeRequest{
			Email:       input.Email,
			AmountMinor: 1,
			MobileMoney: paystack.MobileMoney{
				Phone:   input.Phone,
				Network: paystack.MobileMoneyNetwork(input.Network),
			},
		}).Validate(); err != nil {
			return InitiationResult{}, apperror.NewBadRequest("valid mobile money email, phone and network are required")
		}
	default:
		return InitiationResult{}, apperror.NewBadRequest("unsupported payment provider")
	}

	payment, err := s.CreateForCheckout(ctx, organizationID, checkoutID, input.Provider, attemptKey)
	if err != nil {
		return InitiationResult{}, err
	}
	if input.Provider == "paystack" && payment.Currency != "GHS" {
		return InitiationResult{}, apperror.NewBadRequest("paystack mobile money requires a GHS wallet")
	}

	if payment.Status != "created" || payment.ProviderReference != nil {
		return InitiationResult{}, apperror.NewConflict("payment attempt has already been submitted; check its status")
	}
	if input.Provider == "paystack" {
		return s.initiatePaystack(ctx, payment, input)
	}
	return s.initiateStripe(ctx, payment)
}

func (s *Service) initiateStripe(ctx context.Context, payment sqlc.Payment) (InitiationResult, error) {
	if _, err := s.repo.ClaimStripe(ctx, payment.ID); err != nil {
		return InitiationResult{}, initiationClaimError(err)
	}
	session, err := s.stripeClient.CreateCheckoutSession(ctx, stripe.CreateCheckoutSessionRequest{
		AmountMinor: payment.AmountMinor,
		Currency:    payment.Currency,
		PaymentID:   payment.ID.String(),
	})
	if err != nil {
		return InitiationResult{}, apperror.NewServiceUnavailable(
			"stripe submission is uncertain; check payment status before retrying", err)
	}
	if session.ID == "" || session.ClientSecret == "" ||
		session.AmountTotal != payment.AmountMinor ||
		!strings.EqualFold(session.Currency, payment.Currency) {
		return InitiationResult{}, apperror.NewServiceUnavailable(
			"stripe submission needs reconciliation", nil)
	}
	payment, err = s.repo.SaveStripeRef(ctx, payment.ID, session.ID)
	if err != nil {
		return InitiationResult{}, apperror.NewServiceUnavailable(
			"stripe session was created but its reference needs reconciliation", err)
	}
	return InitiationResult{
		Payment: payment, ClientSecret: session.ClientSecret, ProviderStatus: session.Status,
	}, nil
}

func (s *Service) initiatePaystack(ctx context.Context, payment sqlc.Payment, input CreateInput) (InitiationResult, error) {
	reference := "lm_" + strings.ReplaceAll(payment.ID.String(), "-", "")
	claimed, err := s.repo.ClaimPaystack(ctx, payment.ID, reference)
	if err != nil {
		return InitiationResult{}, initiationClaimError(err)
	}
	charge, err := s.paystackClient.ChargeMobileMoney(ctx, paystack.ChargeRequest{
		Email:       input.Email,
		AmountMinor: payment.AmountMinor,
		Currency:    payment.Currency,
		Reference:   reference,
		MobileMoney: paystack.MobileMoney{
			Phone:   input.Phone,
			Network: paystack.MobileMoneyNetwork(input.Network),
		},
	})
	if err != nil {
		return InitiationResult{}, apperror.NewServiceUnavailable(
			"paystack submission is uncertain; check payment status before retrying", err)
	}
	if charge.Reference != reference {
		return InitiationResult{}, apperror.NewServiceUnavailable(
			"paystack submission reference needs reconciliation", nil)
	}
	return InitiationResult{
		Payment: claimed, DisplayText: charge.DisplayText, ProviderStatus: charge.Status,
	}, nil
}

func initiationClaimError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewConflict("payment is already submitted, or checkout is not available")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.NewConflict("checkout already has an active payment attempt")
	}
	return apperror.NewInternal("claim provider payment attempt", fmt.Errorf("%w", err))
}
