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
	if req.Purpose != PurposeSubscription {
		return fmt.Errorf("%w: purpose must be subscription", ErrInvalidInput)
	}
	if req.SubscriptionID == nil || *req.SubscriptionID == uuid.Nil {
		return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
	}

	return nil
}

func validateConfirmRequest(req *ConfirmRequest) error {
	if req.OrganizationID == uuid.Nil || req.CheckoutID == uuid.Nil {
		return fmt.Errorf("%w: organization and checkout ids are required", ErrInvalidInput)
	}

	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.PaymentMethod = strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if req.Provider != ProviderStripe || req.PaymentMethod != PaymentMethodCard {
		return fmt.Errorf("%w: Stripe card is the only supported payment method", ErrInvalidInput)
	}

	return nil
}
