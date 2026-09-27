package subscriptions

import (
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type CreatePlanRequest struct {
	Code         string
	Name         string
	Currency     string
	AmountMicros int64
}

type CreateRequest struct {
	OrganizationID uuid.UUID
	PlanID         uuid.UUID
}

type PeriodRequest struct {
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	Start          time.Time
	End            time.Time
}

func normalizePlan(req *CreatePlanRequest) error {
	req.Code = strings.ToLower(strings.TrimSpace(req.Code))
	req.Name = strings.TrimSpace(req.Name)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))

	if req.Code == "" || len(req.Code) > 64 {
		return apperror.NewBadRequest("plan code must be between 1 and 64 characters")
	}
	for i, r := range req.Code {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return apperror.NewBadRequest("plan code must use lowercase letters, digits, or underscores")
		}
		if i == 0 && (r < 'a' || r > 'z') {
			return apperror.NewBadRequest("plan code must start with a lowercase letter")
		}
	}
	if req.Name == "" || len(req.Name) > 120 {
		return apperror.NewBadRequest("plan name must be between 1 and 120 characters")
	}
	if len(req.Currency) != 3 {
		return apperror.NewBadRequest("currency must be a 3-letter ISO code")
	}
	for _, r := range req.Currency {
		if r < 'A' || r > 'Z' {
			return apperror.NewBadRequest("currency must be a 3-letter ISO code")
		}
	}
	if req.AmountMicros < 0 {
		return apperror.NewBadRequest("plan amount cannot be negative")
	}
	return nil
}

func validatePeriod(req PeriodRequest) error {
	if req.OrganizationID == uuid.Nil || req.SubscriptionID == uuid.Nil {
		return apperror.NewBadRequest("organization and subscription are required")
	}
	if req.Start.IsZero() || req.End.IsZero() || !req.End.After(req.Start) {
		return apperror.NewBadRequest("subscription period end must be after period start")
	}
	return nil
}
