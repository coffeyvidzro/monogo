package subscriptions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PlanStatusActive   = "active"
	PlanStatusArchived = "archived"

	StatusPending   = "pending"
	StatusActive    = "active"
	StatusPastDue   = "past_due"
	StatusCancelled = "cancelled"
)

var (
	ErrPlanNotFound              = errors.New("subscription plan not found")
	ErrPlanConflict              = errors.New("subscription plan conflict")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionConflict      = errors.New("organization already has a current subscription")
	ErrSubscriptionInvalidState  = errors.New("subscription state does not allow operation")
	ErrSubscriptionNotPermitted  = errors.New("subscription cannot be created")
)

type Plan struct {
	ID           uuid.UUID
	Code         string
	Name         string
	Currency     string
	Interval     string
	AmountMicros int64
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Subscription struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	PlanID             uuid.UUID
	Status             string
	Currency           string
	AmountMicros       int64
	Interval           string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  bool
	StartedAt          *time.Time
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreatePlanRequest struct {
	Code         string
	Name         string
	Currency     string
	AmountMicros int64
}

type SubscribeRequest struct {
	OrganizationID uuid.UUID
	PlanID         uuid.UUID
}

type ActivateRequest struct {
	OrganizationID     uuid.UUID
	SubscriptionID     uuid.UUID
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}
