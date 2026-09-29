package payments

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending           = "pending"
	StatusProcessing        = "processing"
	StatusSucceeded         = "succeeded"
	StatusFailed            = "failed"
	StatusCancelled         = "cancelled"
	StatusRefunded          = "refunded"
	StatusPartiallyRefunded = "partially_refunded"
)

var (
	ErrInvalidInput = errors.New("invalid payment input")
	ErrNotFound     = errors.New("payment not found")
	ErrConflict     = errors.New("payment state does not allow operation")
)

type Payment struct {
	ID                uuid.UUID
	CheckoutID        uuid.UUID
	OrganizationID    uuid.UUID
	Provider          string
	Attempt           int32
	ProviderPaymentID *string
	AmountMicros      int64
	Currency          string
	Status            string
	FailureCode       *string
	PaidAt            *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateAttemptRequest struct {
	CheckoutID     uuid.UUID
	OrganizationID uuid.UUID
	Provider       string
	AmountMicros   int64
	Currency       string
}

type AttachProviderPaymentIDRequest struct {
	CheckoutID        uuid.UUID
	OrganizationID    uuid.UUID
	PaymentID          uuid.UUID
	ProviderPaymentID string
}
