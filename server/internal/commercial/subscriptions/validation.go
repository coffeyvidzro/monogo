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
		return fmt.Errorf("plan code is invalid")
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 1 || len(req.Name) > 120 {
		return fmt.Errorf("plan name must be between 1 and 120 characters")
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("currency must be a three-letter code")
	}

	if req.AmountMicros < 0 {
		return fmt.Errorf("plan amount cannot be negative")
	}

	return nil
}

func validateSubscribeRequest(req SubscribeRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if req.PlanID == uuid.Nil {
		return fmt.Errorf("plan id is required")
	}

	return nil
}

func validateActivateRequest(req ActivateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if req.SubscriptionID == uuid.Nil {
		return fmt.Errorf("subscription id is required")
	}
	if req.CurrentPeriodStart.IsZero() {
		return fmt.Errorf("current period start is required")
	}
	if req.CurrentPeriodEnd.IsZero() {
		return fmt.Errorf("current period end is required")
	}
	if !req.CurrentPeriodEnd.After(req.CurrentPeriodStart) {
		return fmt.Errorf("current period end must follow current period start")
	}

	return nil
}
