package checkout

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PurposeSubscription = "subscription"
	PurposeWalletTopup  = "wallet_topup"

	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSucceeded  = "succeeded"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
	StatusExpired    = "expired"

	ActionNone = "none"
	ActionWait = "wait"

	ProviderStripe = "stripe"

	PaymentMethodCard = "card"
)

var ErrInvalidInput = errors.New("invalid checkout input")

type Checkout struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organization_id"`
	Purpose           string     `json:"purpose"`
	SubscriptionID    *uuid.UUID `json:"subscription_id,omitempty"`
	Reference         string     `json:"reference"`
	AmountMicros      int64      `json:"amount_micros"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	Provider          *string    `json:"provider,omitempty"`
	PaymentMethod     *string    `json:"payment_method,omitempty"`
	NextAction        string     `json:"next_action"`
	ProviderMessage   *string    `json:"provider_message,omitempty"`
	FailureCode       *string    `json:"failure_code,omitempty"`
	ExpiresAt         time.Time  `json:"expires_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ClientSecret      *string    `json:"client_secret,omitempty"`
	ProviderPaymentID *string    `json:"provider_payment_id,omitempty"`
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
	PaymentMethod  string `json:"payment_method"`
}
