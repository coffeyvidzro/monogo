package payments

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func validateCreateAttemptRequest(req *CreateAttemptRequest) error {
	if req.CheckoutID == uuid.Nil || req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: checkout and organization ids are required", ErrInvalidInput)
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.PaymentMethod = strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if req.Provider != "stripe" || req.PaymentMethod != "card" {
		return fmt.Errorf("%w: Stripe card is the only supported payment method", ErrInvalidInput)
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency != "USD" {
		return fmt.Errorf("%w: USD is the only supported payment currency", ErrInvalidInput)
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}
	return nil
}

func validateAttachProviderPaymentIDRequest(req *AttachProviderPaymentIDRequest) error {
	if req.CheckoutID == uuid.Nil || req.OrganizationID == uuid.Nil || req.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: checkout, organization, and payment ids are required", ErrInvalidInput)
	}
	req.ProviderPaymentID = strings.TrimSpace(req.ProviderPaymentID)
	if req.ProviderPaymentID == "" {
		return fmt.Errorf("%w: provider payment id is required", ErrInvalidInput)
	}
	return nil
}

func validateRecordProviderEventRequest(req *RecordProviderEventRequest) error {
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.ProviderPaymentID = strings.TrimSpace(req.ProviderPaymentID)
	req.ProviderEventID = strings.TrimSpace(req.ProviderEventID)
	req.EventType = strings.TrimSpace(req.EventType)
	if req.Provider != "stripe" {
		return fmt.Errorf("%w: Stripe is the only supported payment provider", ErrInvalidInput)
	}
	if req.ProviderPaymentID == "" {
		return fmt.Errorf("%w: provider payment id is required", ErrInvalidInput)
	}
	if req.ProviderEventID == "" {
		return fmt.Errorf("%w: provider event id is required", ErrInvalidInput)
	}
	if req.EventType == "" {
		return fmt.Errorf("%w: event type is required", ErrInvalidInput)
	}
	if len(req.Payload) == 0 {
		return fmt.Errorf("%w: payload is required", ErrInvalidInput)
	}
	return nil
}
