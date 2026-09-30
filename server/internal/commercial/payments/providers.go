package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type StartRequest struct {
	Payment       Payment
	Reference     string
	Purpose       string
	Email         string
	Phone         string
	MobileNetwork string
}

type StartResult struct {
	ProviderPaymentID string
	ClientSecret      *string
	NextAction        string
	ProviderMessage   *string
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

func (s *ProviderService) Start(
	ctx context.Context,
	req StartRequest,
) (StartResult, error) {
	amountMinor, err := microsToMinor(req.Payment.AmountMicros)
	if err != nil {
		return StartResult{}, err
	}

	switch req.Payment.Provider {
	case "stripe":
		if s == nil || s.Stripe == nil {
			return StartResult{}, fmt.Errorf("stripe provider is not configured")
		}
		productName := "Monogo subscription"
		if req.Purpose == "wallet_topup" {
			productName = "Wallet top-up"
		}
		session, err := s.Stripe.CreateCheckoutSession(
			ctx,
			stripe.CreateCheckoutSessionRequest{
				AmountMinor:    amountMinor,
				Currency:       req.Payment.Currency,
				Reference:      req.Reference,
				IdempotencyKey: req.Payment.ID.String(),
				ProductName:    productName,
			},
		)
		if err != nil {
			return StartResult{}, fmt.Errorf("create Stripe checkout session: %w", err)
		}
		if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.ClientSecret) == "" {
			return StartResult{}, fmt.Errorf("stripe checkout session is incomplete")
		}
		if session.AmountTotal != amountMinor ||
			strings.ToUpper(session.Currency) != req.Payment.Currency {
			return StartResult{}, fmt.Errorf("stripe checkout amount or currency does not match payment")
		}
		clientSecret := session.ClientSecret
		return StartResult{
			ProviderPaymentID: session.ID,
			ClientSecret:      &clientSecret,
			NextAction:        "wait",
		}, nil

	case "paystack":
		if s == nil || s.Paystack == nil {
			return StartResult{}, fmt.Errorf("paystack provider is not configured")
		}
		charge, err := s.Paystack.ChargeMobileMoney(
			ctx,
			paystack.ChargeRequest{
				Email:       req.Email,
				AmountMinor: amountMinor,
				Currency:    req.Payment.Currency,
				Reference:   req.Reference,
				MobileMoney: paystack.MobileMoney{
					Phone:   req.Phone,
					Network: paystack.MobileMoneyNetwork(req.MobileNetwork),
				},
			},
		)
		if err != nil {
			return StartResult{}, fmt.Errorf("create Paystack mobile money charge: %w", err)
		}
		if charge.Reference != req.Reference {
			return StartResult{}, fmt.Errorf("paystack returned an unexpected charge reference")
		}
		state := paystack.MapCharge(charge)
		message := strings.TrimSpace(state.Message)
		var providerMessage *string
		if message != "" {
			providerMessage = &message
		}
		return StartResult{
			ProviderPaymentID: charge.Reference,
			NextAction:        state.NextAction,
			ProviderMessage:   providerMessage,
		}, nil

	default:
		return StartResult{}, fmt.Errorf("unsupported payment provider %q", req.Payment.Provider)
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

type ProviderWebhook struct {
	Provider          string
	ProviderPaymentID string
	ProviderEventID   string
	EventType         string
}

func (s *ProviderService) ParseStripeWebhook(
	payload []byte,
	signature string,
	now time.Time,
) (ProviderWebhook, error) {
	if s == nil || s.Stripe == nil {
		return ProviderWebhook{}, fmt.Errorf("stripe provider is not configured")
	}

	event, err := s.Stripe.ParseWebhook(payload, signature, now)
	if err != nil {
		return ProviderWebhook{}, err
	}
	session, err := event.DecodeCheckoutSession()
	if err != nil {
		return ProviderWebhook{}, err
	}

	return ProviderWebhook{
		Provider:          "stripe",
		ProviderPaymentID: session.ID,
		ProviderEventID:   event.ID,
		EventType:         event.Type,
	}, nil
}

func (s *ProviderService) ParsePaystackWebhook(
	payload []byte,
	signature string,
) (ProviderWebhook, error) {
	if s == nil || s.Paystack == nil {
		return ProviderWebhook{}, fmt.Errorf("paystack provider is not configured")
	}

	event, err := s.Paystack.ParseWebhook(payload, signature)
	if err != nil {
		return ProviderWebhook{}, err
	}

	sum := sha256.Sum256(payload)
	return ProviderWebhook{
		Provider:          "paystack",
		ProviderPaymentID: event.Data.Reference,
		ProviderEventID:   event.Event + ":" + event.Data.Reference + ":" + hex.EncodeToString(sum[:]),
		EventType:         event.Event,
	}, nil
}

func IsSuccessfulProviderEvent(provider, eventType string) bool {
	switch provider {
	case "stripe":
		return eventType == "checkout.session.completed" ||
			eventType == "checkout.session.async_payment_succeeded"
	case "paystack":
		return eventType == "charge.success"
	default:
		return false
	}
}
