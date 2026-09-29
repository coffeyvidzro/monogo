package payments

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	providerPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

func validateCreateRequest(req *CreateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}

	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if !providerPattern.MatchString(req.Provider) {
		return fmt.Errorf("%w: provider is invalid", ErrInvalidInput)
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if !currencyPattern.MatchString(req.Currency) {
		return fmt.Errorf("%w: currency is invalid", ErrInvalidInput)
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}

	switch req.Purpose {
	case PurposeSubscription:
		if req.SubscriptionID == nil || *req.SubscriptionID == uuid.Nil {
			return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
		}
		if req.PeriodStart == nil || req.PeriodEnd == nil || !req.PeriodEnd.After(*req.PeriodStart) {
			return fmt.Errorf("%w: valid subscription period is required", ErrInvalidInput)
		}
	case PurposeWalletTopup:
		if req.SubscriptionID != nil || req.PeriodStart != nil || req.PeriodEnd != nil {
			return fmt.Errorf("%w: wallet top-up cannot include subscription fields", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: purpose is invalid", ErrInvalidInput)
	}

	return nil
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

func validateSucceedRequest(req *SucceedRequest) error {
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
