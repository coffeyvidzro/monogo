package payments

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PurposeSubscription = "subscription"
	PurposeWalletTopup  = "wallet_topup"

	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

var (
	ErrInvalidInput = errors.New("invalid payment input")
	ErrNotFound     = errors.New("payment not found")
	ErrConflict     = errors.New("payment state does not allow operation")
)

type Payment struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organization_id"`
	Purpose           string     `json:"purpose"`
	Provider          string     `json:"provider"`
	SubscriptionID    *uuid.UUID `json:"subscription_id,omitempty"`
	AmountMicros      int64      `json:"amount_micros"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	ProviderReference *string    `json:"provider_reference,omitempty"`
	ProviderEventID   *string    `json:"provider_event_id,omitempty"`
	PeriodStart       *time.Time `json:"period_start,omitempty"`
	PeriodEnd         *time.Time `json:"period_end,omitempty"`
	FailureCode       *string    `json:"failure_code,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	OrganizationID uuid.UUID
	Purpose        string
	Provider       string
	SubscriptionID *uuid.UUID
	AmountMicros   int64
	Currency       string
	PeriodStart    *time.Time
	PeriodEnd      *time.Time
}

type AttachProviderReferenceRequest struct {
	OrganizationID    uuid.UUID
	PaymentID         uuid.UUID
	ProviderReference string
}

type SucceedRequest struct {
	OrganizationID  uuid.UUID
	PaymentID       uuid.UUID
	ProviderEventID string
	OccurredAt      time.Time
}

type FailRequest struct {
	OrganizationID  uuid.UUID
	PaymentID       uuid.UUID
	ProviderEventID string
	FailureCode     string
	OccurredAt      time.Time
}
