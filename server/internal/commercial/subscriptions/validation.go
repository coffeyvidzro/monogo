package subscriptions

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var planCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func normalizeCreatePlanRequest(req *CreatePlanRequest) error {
	req.Code = strings.ToLower(strings.TrimSpace(req.Code))
	if !planCodePattern.MatchString(req.Code) {
		return fmt.Errorf("%w: plan code is invalid", ErrInvalidInput)
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 1 || len(req.Name) > 120 {
		return fmt.Errorf("%w: plan name must be between 1 and 120 characters", ErrInvalidInput)
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("%w: currency must be a three-letter code", ErrInvalidInput)
	}

	if req.AmountMicros < 0 {
		return fmt.Errorf("%w: plan amount cannot be negative", ErrInvalidInput)
	}

	return nil
}

func validateSubscribeRequest(req SubscribeRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.PlanID == uuid.Nil {
		return fmt.Errorf("%w: plan id is required", ErrInvalidInput)
	}

	return nil
}

func validateActivateRequest(req ActivateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.SubscriptionID == uuid.Nil {
		return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
	}
	if req.CurrentPeriodStart.IsZero() {
		return fmt.Errorf("%w: current period start is required", ErrInvalidInput)
	}
	if req.CurrentPeriodEnd.IsZero() {
		return fmt.Errorf("%w: current period end is required", ErrInvalidInput)
	}
	if !req.CurrentPeriodEnd.After(req.CurrentPeriodStart) {
		return fmt.Errorf("%w: current period end must follow current period start", ErrInvalidInput)
	}

	return nil
}
