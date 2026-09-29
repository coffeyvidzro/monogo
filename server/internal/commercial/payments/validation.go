package payments

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var providerPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func normalizeProvider(value *string) error {
	*value = strings.ToLower(strings.TrimSpace(*value))
	if !providerPattern.MatchString(*value) {
		return fmt.Errorf("%w: provider is invalid", ErrInvalidInput)
	}
	return nil
}

func validateCreateSubscriptionRequest(req *CreateSubscriptionRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.SubscriptionID == uuid.Nil {
		return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
	}
	return normalizeProvider(&req.Provider)
}

func validateCreateWalletTopupRequest(req *CreateWalletTopupRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}
	return normalizeProvider(&req.Provider)
}

func validateAttachProviderReferenceRequest(req *AttachProviderReferenceRequest) error {
	if req.OrganizationID == uuid.Nil || req.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: organization and payment ids are required", ErrInvalidInput)
	}
	req.ProviderReference = strings.TrimSpace(req.ProviderReference)
	if req.ProviderReference == "" {
		return fmt.Errorf("%w: provider reference is required", ErrInvalidInput)
	}
	return nil
}

func validateSettleRequest(req *SettleRequest) error {
	if req.OrganizationID == uuid.Nil || req.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: organization and payment ids are required", ErrInvalidInput)
	}
	req.ProviderEventID = strings.TrimSpace(req.ProviderEventID)
	if req.ProviderEventID == "" {
		return fmt.Errorf("%w: provider event id is required", ErrInvalidInput)
	}
	return nil
}

func validateFailRequest(req *FailRequest) error {
	if req.OrganizationID == uuid.Nil || req.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: organization and payment ids are required", ErrInvalidInput)
	}
	req.ProviderEventID = strings.TrimSpace(req.ProviderEventID)
	req.FailureCode = strings.TrimSpace(req.FailureCode)
	if req.ProviderEventID == "" {
		return fmt.Errorf("%w: provider event id is required", ErrInvalidInput)
	}
	if req.FailureCode == "" {
		return fmt.Errorf("%w: failure code is required", ErrInvalidInput)
	}
	return nil
}
