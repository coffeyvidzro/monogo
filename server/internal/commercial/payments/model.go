package payments

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
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
	ID                uuid.UUID  `json:"-"`
	CheckoutID        uuid.UUID  `json:"-"`
	OrganizationID    uuid.UUID  `json:"-"`
	Provider          string     `json:"-"`
	Attempt           int32      `json:"-"`
	AmountMicros      int64      `json:"-"`
	Currency          string     `json:"-"`
	Status            string     `json:"-"`
	ProviderReference *string    `json:"-"`
	FailureCode       *string    `json:"-"`
	CompletedAt       *time.Time `json:"-"`
	CreatedAt         time.Time  `json:"-"`
	UpdatedAt         time.Time  `json:"-"`
}

type CreateAttemptRequest struct {
	CheckoutID     uuid.UUID
	OrganizationID uuid.UUID
	Provider       string
	AmountMicros   int64
	Currency       string
}

type AttachProviderReferenceRequest struct {
	CheckoutID       uuid.UUID
	OrganizationID   uuid.UUID
	PaymentID         uuid.UUID
	ProviderReference string
}
