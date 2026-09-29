package checkout

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PurposeSubscription = "subscription"
	PurposeWalletTopup   = "wallet_topup"

	StatusOpen       = "open"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

var ErrInvalidInput = errors.New("invalid checkout input")

type Checkout struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Purpose        string     `json:"purpose"`
	Status         string     `json:"status"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	AmountMicros   int64      `json:"amount_micros"`
	Currency       string     `json:"currency"`
	PeriodStart    *time.Time `json:"period_start,omitempty"`
	PeriodEnd      *time.Time `json:"period_end,omitempty"`
	FailureCode    *string    `json:"failure_code,omitempty"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	FailedAt       *time.Time `json:"failed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	OrganizationID uuid.UUID
	Purpose        string     `json:"purpose"`
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"`
	AmountMicros   int64      `json:"amount_micros,omitempty"`
}

type ConfirmRequest struct {
	OrganizationID uuid.UUID
	CheckoutID     uuid.UUID
	Provider       string `json:"provider"`
}

type ContinueRequest struct {
	OrganizationID uuid.UUID
	CheckoutID     uuid.UUID
	Provider       string `json:"provider"`
}
