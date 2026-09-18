package paystack

import "strings"

type PaymentStatus string

const (
	PaymentStatusPending        PaymentStatus = "pending"
	PaymentStatusRequiresAction PaymentStatus = "requires_action"
	PaymentStatusProcessing     PaymentStatus = "processing"
	PaymentStatusSucceeded      PaymentStatus = "succeeded"
	PaymentStatusFailed         PaymentStatus = "failed"
)

func MapStatus(status string) PaymentStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return PaymentStatusSucceeded
	case "pay_offline", "pending", "ongoing", "processing", "queued":
		return PaymentStatusProcessing
	case "send_otp", "send_pin", "send_phone", "send_birthday", "send_address", "open_url":
		return PaymentStatusRequiresAction
	case "failed", "declined", "abandoned", "reversed", "timeout":
		return PaymentStatusFailed
	default:
		return PaymentStatusPending
	}
}
