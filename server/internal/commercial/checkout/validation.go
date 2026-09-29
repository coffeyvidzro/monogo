package checkout

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func validateCreateRequest(req *CreateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}

	req.Purpose = strings.ToLower(strings.TrimSpace(req.Purpose))
	switch req.Purpose {
	case PurposeSubscription:
		if req.SubscriptionID == nil || *req.SubscriptionID == uuid.Nil {
			return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
		}
		if req.AmountMicros != 0 {
			return fmt.Errorf("%w: subscription amount is derived by checkout", ErrInvalidInput)
		}
	case PurposeWalletTopup:
		if req.SubscriptionID != nil {
			return fmt.Errorf("%w: wallet top-up cannot include subscription id", ErrInvalidInput)
		}
		if req.AmountMicros <= 0 {
			return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: purpose is invalid", ErrInvalidInput)
	}

	return nil
}

func validateConfirmRequest(req *ConfirmRequest) error {
	if req.OrganizationID == uuid.Nil || req.CheckoutID == uuid.Nil {
		return fmt.Errorf("%w: organization and checkout ids are required", ErrInvalidInput)
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if req.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	return nil
}

func validateContinueRequest(req *ContinueRequest) error {
	if req.OrganizationID == uuid.Nil || req.CheckoutID == uuid.Nil {
		return fmt.Errorf("%w: organization and checkout ids are required", ErrInvalidInput)
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if req.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidInput)
	}
	return nil
}
