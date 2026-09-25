package worker

import (
	"context"
	"strings"
	"time"

	commercialpayments "github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
)

type stripePaymentVerifier struct{ client *stripe.Client }

func (v stripePaymentVerifier) Verify(ctx context.Context, candidate commercialpayments.RecoveryCandidate) (commercialpayments.Verification, error) {
	session, err := v.client.RetrieveCheckoutSession(ctx, candidate.ProviderReference)
	if err != nil {
		return commercialpayments.Verification{}, err
	}
	return commercialpayments.Verification{Succeeded: session.PaymentStatus == "paid",
		AmountMinor: session.AmountTotal, Currency: strings.ToUpper(session.Currency), VerifiedAt: time.Now().UTC()}, nil
}

type paystackPaymentVerifier struct{ client *paystack.Client }

func (v paystackPaymentVerifier) Verify(ctx context.Context, candidate commercialpayments.RecoveryCandidate) (commercialpayments.Verification, error) {
	transaction, err := v.client.VerifyTransaction(ctx, candidate.ProviderReference)
	if err != nil {
		return commercialpayments.Verification{}, err
	}
	return commercialpayments.Verification{Succeeded: transaction.Status == "success",
		AmountMinor: transaction.AmountMinor, Currency: strings.ToUpper(transaction.Currency), VerifiedAt: time.Now().UTC()}, nil
}

func paymentVerifiers(stripeClient *stripe.Client, paystackClient *paystack.Client) map[string]commercialpayments.ProviderVerifier {
	result := make(map[string]commercialpayments.ProviderVerifier, 2)
	if stripeClient != nil {
		result["stripe"] = stripePaymentVerifier{client: stripeClient}
	}
	if paystackClient != nil {
		result["paystack"] = paystackPaymentVerifier{client: paystackClient}
	}
	return result
}
