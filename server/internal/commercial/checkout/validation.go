package checkout

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func validateCreateSubscriptionRequest(req CreateSubscriptionRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.SubscriptionID == uuid.Nil {
		return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Provider) == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}

	return nil
}

func validateCreateWalletTopupRequest(req CreateWalletTopupRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Provider) == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}

	return nil
}

func validateCompleteRequest(req CompleteRequest) error {
	if req.OrganizationID == uuid.Nil || req.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: organization and payment ids are required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.ProviderEventID) == "" {
		return fmt.Errorf("%w: provider event id is required", ErrInvalidInput)
	}

	return nil
}

func validateFailRequest(req FailRequest) error {
	if req.OrganizationID == uuid.Nil || req.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: organization and payment ids are required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.ProviderEventID) == "" {
		return fmt.Errorf("%w: provider event id is required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.FailureCode) == "" {
		return fmt.Errorf("%w: failure code is required", ErrInvalidInput)
	}

	return nil
}
