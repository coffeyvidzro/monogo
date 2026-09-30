package stripe

import "strings"

type CheckoutState struct {
	Status      string
	NextAction  string
	Succeeded   bool
	Failed      bool
	FailureCode string
}

func MapCheckoutSession(session CheckoutSession) CheckoutState {
	status := strings.ToLower(strings.TrimSpace(session.Status))
	paymentStatus := strings.ToLower(strings.TrimSpace(session.PaymentStatus))

	switch {
	case status == "complete" && paymentStatus == "paid":
		return CheckoutState{
			Status:     "succeeded",
			NextAction: "none",
			Succeeded:  true,
		}
	case status == "expired":
		return CheckoutState{
			Status:      "failed",
			NextAction:  "none",
			Failed:      true,
			FailureCode: "expired",
		}
	default:
		return CheckoutState{
			Status:     "processing",
			NextAction: "wait",
		}
	}
}
