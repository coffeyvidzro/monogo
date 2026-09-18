package paystack

import "strings"

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
)

func MapStatus(status string) PaymentStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success":
		return PaymentStatusSucceeded
	case "pay_offline", "pending":
		return PaymentStatusProcessing
	case "failed", "abandoned":
		return PaymentStatusFailed
	default:
		return PaymentStatusPending
	}
}
