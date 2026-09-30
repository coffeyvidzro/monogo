package paystack

import "strings"

type ChargeState struct {
	Status      string
	NextAction  string
	Message     string
	Succeeded   bool
	Failed      bool
	FailureCode string
}

func MapCharge(charge Charge) ChargeState {
	status := strings.ToLower(strings.TrimSpace(charge.Status))
	message := strings.TrimSpace(charge.DisplayText)

	switch status {
	case "success":
		return ChargeState{
			Status:     status,
			NextAction: "none",
			Message:    message,
			Succeeded:  true,
		}
	case "pending":
		return ChargeState{
			Status:     status,
			NextAction: "wait",
			Message:    message,
		}
	case "pay_offline":
		return ChargeState{
			Status:     status,
			NextAction: "authorize_mobile_money",
			Message:    message,
		}
	case "send_otp":
		return ChargeState{
			Status:     status,
			NextAction: "submit_otp",
			Message:    message,
		}
	case "send_phone":
		return ChargeState{
			Status:     status,
			NextAction: "submit_phone",
			Message:    message,
		}
	case "failed", "timeout":
		return ChargeState{
			Status:      status,
			NextAction:  "none",
			Message:     message,
			Failed:      true,
			FailureCode: status,
		}
	default:
		return ChargeState{
			Status:     status,
			NextAction: "unsupported",
			Message:    message,
		}
	}
}
