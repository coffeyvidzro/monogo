package stripe

import "strings"

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
)

func MapCheckoutStatus(session CheckoutSession) PaymentStatus {
	switch strings.ToLower(strings.TrimSpace(session.PaymentStatus)) {
	case "paid", "no_payment_required":
		return PaymentStatusSucceeded
	}

	switch strings.ToLower(strings.TrimSpace(session.Status)) {
	case "complete":
		return PaymentStatusProcessing
	case "expired":
		return PaymentStatusFailed
	default:
		return PaymentStatusPending
	}
}
