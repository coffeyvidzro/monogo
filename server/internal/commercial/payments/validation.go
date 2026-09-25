package payments

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func validateAttempt(req *Attempt) error {
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.AttemptKey = strings.TrimSpace(req.AttemptKey)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.OrganizationID == uuid.Nil || req.CheckoutID == uuid.Nil || req.AmountMinor <= 0 {
		return apperror.NewBadRequest("organization, checkout, and positive amount are required")
	}
	if req.Provider != "stripe" && req.Provider != "paystack" {
		return apperror.NewBadRequest("unsupported payment provider")
	}
	if len(req.AttemptKey) == 0 || len(req.AttemptKey) > 255 {
		return apperror.NewBadRequest("payment attempt key is required")
	}
	if len(req.Currency) != 3 {
		return apperror.NewBadRequest("payment currency must be a three-letter code")
	}
	for _, ch := range req.Currency {
		if ch < 'A' || ch > 'Z' {
			return apperror.NewBadRequest("payment currency must be a three-letter code")
		}
	}
	return nil
}

func validateEvent(req *Event) error {
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.ProviderEventID = strings.TrimSpace(req.ProviderEventID)
	req.EventType = strings.TrimSpace(req.EventType)
	if req.Provider != "stripe" && req.Provider != "paystack" {
		return apperror.NewBadRequest("unsupported payment provider")
	}
	if req.ProviderEventID == "" || req.EventType == "" {
		return apperror.NewBadRequest("payment event identity and type are required")
	}
	if req.PaymentID != nil && *req.PaymentID == uuid.Nil {
		return apperror.NewBadRequest("invalid payment ID")
	}
	if len(req.PayloadSHA256) != 64 {
		return apperror.NewBadRequest("payment event payload digest is required")
	}
	for _, ch := range req.PayloadSHA256 {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return apperror.NewBadRequest("invalid payment event payload digest")
		}
	}
	return nil
}
