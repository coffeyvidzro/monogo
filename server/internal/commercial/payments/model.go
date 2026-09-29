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

type CreateSubscriptionRequest struct {
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	Provider       string
}

type CreateWalletTopupRequest struct {
	OrganizationID uuid.UUID
	Provider       string
	AmountMicros   int64
}

type AttachProviderReferenceRequest struct {
	OrganizationID    uuid.UUID
	PaymentID         uuid.UUID
	ProviderReference string
}

type SettleRequest struct {
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
