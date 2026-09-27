package subscriptions

import (
	"time"

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
