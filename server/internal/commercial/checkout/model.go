package checkout

import (
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid checkout input")

type CreateSubscriptionRequest struct {
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Provider       string    `json:"provider"`
}

type CreateWalletTopupRequest struct {
	OrganizationID uuid.UUID
	Provider       string `json:"provider"`
	AmountMicros   int64  `json:"amount_micros"`
}

type CompleteRequest struct {
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

type Checkout struct {
	Payment payments.Payment `json:"payment"`
}
