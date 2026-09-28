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
	ErrPlanNotFound             = errors.New("subscription plan not found")
	ErrPlanConflict             = errors.New("subscription plan conflict")
	ErrSubscriptionNotFound     = errors.New("subscription not found")
	ErrSubscriptionConflict     = errors.New("organization already has a current subscription")
	ErrSubscriptionInvalidState = errors.New("subscription state does not allow operation")
	ErrSubscriptionNotPermitted = errors.New("subscription cannot be created")
)

type Plan struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Currency     string    `json:"currency"`
	Interval     string    `json:"interval"`
	AmountMicros int64     `json:"amount_micros"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Subscription struct {
	ID                 uuid.UUID  `json:"id"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	PlanID             uuid.UUID  `json:"plan_id"`
	Status             string     `json:"status"`
	Currency           string     `json:"currency"`
	AmountMicros       int64      `json:"amount_micros"`
	Interval           string     `json:"interval"`
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool       `json:"cancel_at_period_end"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CancelledAt        *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
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
