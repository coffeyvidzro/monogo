package payments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
)

// StripeVerifier checks an existing, durably referenced provider payment.
// It never treats a browser redirect or a client-submitted status as settlement.
type StripeVerifier struct {
	Client *stripe.Client
}

func (v StripeVerifier) Verify(ctx context.Context, candidate RecoveryCandidate) (Verification, error) {
	if v.Client == nil || candidate.Provider != "stripe" || candidate.ProviderReference == "" {
		return Verification{}, fmt.Errorf("stripe verification requires a configured provider and a persisted reference")
	}
	session, err := v.Client.RetrieveCheckoutSession(ctx, candidate.ProviderReference)
	if err != nil {
		return Verification{}, fmt.Errorf("retrieve Stripe checkout session: %w", err)
	}
	if session.ID != candidate.ProviderReference {
		return Verification{}, fmt.Errorf("stripe returned an unexpected checkout session")
	}
	if session.PaymentStatus != "paid" || session.Status != "complete" {
		return Verification{}, nil
	}
	if session.AmountTotal != candidate.AmountMinor ||
		strings.ToUpper(session.Currency) != candidate.Currency {
		return Verification{}, fmt.Errorf("stripe settlement amount or currency does not match the payment attempt")
	}
	return Verification{
		Succeeded:   true,
		AmountMinor: session.AmountTotal,
		Currency:    candidate.Currency,
		VerifiedAt:  time.Now().UTC(),
	}, nil
}

// PaystackVerifier independently retrieves a transaction by the reference
// recorded on the attempt before authorizing a wallet credit.
type PaystackVerifier struct {
	Client *paystack.Client
}

func (v PaystackVerifier) Verify(ctx context.Context, candidate RecoveryCandidate) (Verification, error) {
	if v.Client == nil || candidate.Provider != "paystack" || candidate.ProviderReference == "" {
		return Verification{}, fmt.Errorf("paystack verification requires a configured provider and a persisted reference")
	}
	transaction, err := v.Client.VerifyTransaction(ctx, candidate.ProviderReference)
	if err != nil {
		return Verification{}, fmt.Errorf("verify Paystack transaction: %w", err)
	}
	if transaction.Reference != candidate.ProviderReference {
		return Verification{}, fmt.Errorf("paystack returned an unexpected transaction reference")
	}
	if transaction.Status != "success" {
		return Verification{}, nil
	}
	if transaction.AmountMinor != candidate.AmountMinor ||
		strings.ToUpper(transaction.Currency) != candidate.Currency {
		return Verification{}, fmt.Errorf("paystack settlement amount or currency does not match the payment attempt")
	}
	return Verification{
		Succeeded:   true,
		AmountMinor: transaction.AmountMinor,
		Currency:    candidate.Currency,
		VerifiedAt:  time.Now().UTC(),
	}, nil
}

// NewProviderVerifiers exposes only providers with configured credentials.
// This is the commercial boundary for trusted provider result verification;
// provider clients and credentials are never exposed by customer routes.
func NewProviderVerifiers(stripeClient *stripe.Client, paystackClient *paystack.Client) map[string]ProviderVerifier {
	verifiers := make(map[string]ProviderVerifier, 2)
	if stripeClient != nil {
		verifiers["stripe"] = StripeVerifier{Client: stripeClient}
	}
	if paystackClient != nil {
		verifiers["paystack"] = PaystackVerifier{Client: paystackClient}
	}
	return verifiers
}
