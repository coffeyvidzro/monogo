package payments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
)

type ProviderService struct {
	Stripe   *stripe.Client
	Paystack *paystack.Client
}

func NewProviderService(
	stripeClient *stripe.Client,
	paystackClient *paystack.Client,
) *ProviderService {
	return &ProviderService{
		Stripe:   stripeClient,
		Paystack: paystackClient,
	}
}

func (s *ProviderService) Verify(
	ctx context.Context,
	payment Payment,
) (time.Time, error) {
	if payment.ProviderPaymentID == nil || strings.TrimSpace(*payment.ProviderPaymentID) == "" {
		return time.Time{}, fmt.Errorf("payment provider id is required")
	}

	amountMinor, err := microsToMinor(payment.AmountMicros)
	if err != nil {
		return time.Time{}, err
	}

	switch payment.Provider {
	case "stripe":
		if s == nil || s.Stripe == nil {
			return time.Time{}, fmt.Errorf("stripe provider is not configured")
		}
		session, err := s.Stripe.RetrieveCheckoutSession(ctx, *payment.ProviderPaymentID)
		if err != nil {
			return time.Time{}, fmt.Errorf("retrieve Stripe checkout session: %w", err)
		}
		if session.ID != *payment.ProviderPaymentID {
			return time.Time{}, fmt.Errorf("stripe returned an unexpected checkout session")
		}
		if session.Status != "complete" || session.PaymentStatus != "paid" {
			return time.Time{}, fmt.Errorf("stripe payment is not settled")
		}
		if session.AmountTotal != amountMinor ||
			strings.ToUpper(session.Currency) != payment.Currency {
			return time.Time{}, fmt.Errorf("stripe settlement amount or currency does not match payment")
		}

	case "paystack":
		if s == nil || s.Paystack == nil {
			return time.Time{}, fmt.Errorf("paystack provider is not configured")
		}
		transaction, err := s.Paystack.VerifyTransaction(ctx, *payment.ProviderPaymentID)
		if err != nil {
			return time.Time{}, fmt.Errorf("verify Paystack transaction: %w", err)
		}
		if transaction.Reference != *payment.ProviderPaymentID {
			return time.Time{}, fmt.Errorf("paystack returned an unexpected transaction reference")
		}
		if transaction.Status != "success" {
			return time.Time{}, fmt.Errorf("paystack payment is not settled")
		}
		if transaction.AmountMinor != amountMinor ||
			strings.ToUpper(transaction.Currency) != payment.Currency {
			return time.Time{}, fmt.Errorf("paystack settlement amount or currency does not match payment")
		}

	default:
		return time.Time{}, fmt.Errorf("unsupported payment provider %q", payment.Provider)
	}

	return time.Now().UTC(), nil
}

func microsToMinor(amountMicros int64) (int64, error) {
	const microsPerMinor int64 = 10_000
	if amountMicros <= 0 || amountMicros%microsPerMinor != 0 {
		return 0, fmt.Errorf("amount cannot be represented in provider minor units")
	}
	return amountMicros / microsPerMinor, nil
}
