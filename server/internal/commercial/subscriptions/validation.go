package subscriptions

import (
	"fmt"

	"github.com/google/uuid"
)

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

func validateUpdateRequest(req UpdateRequest) error {
	if req.CancelAtPeriodEnd == nil {
		return fmt.Errorf("%w: cancel_at_period_end is required", ErrInvalidInput)
	}

	return nil
}

func normalizeListRequest(req *ListRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.Offset < 0 {
		return fmt.Errorf("%w: offset cannot be negative", ErrInvalidInput)
	}
	if req.Limit == 0 {
		req.Limit = 50
	}
	if req.Limit < 1 || req.Limit > 100 {
		return fmt.Errorf("%w: limit must be between 1 and 100", ErrInvalidInput)
	}

	return nil
}
