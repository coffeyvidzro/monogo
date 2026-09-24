package payments

import "github.com/google/uuid"

// Attempt is one durable attempt to collect a wallet top-up checkout.
type Attempt struct {
	OrganizationID uuid.UUID
	CheckoutID     uuid.UUID
	Provider       string
	AttemptKey     string
	AmountMinor    int64
	Currency       string
}

// Event represents metadata from an already authenticated provider webhook.
// Event persistence does not imply payment verification or wallet crediting.
type Event struct {
	PaymentID       *uuid.UUID
	Provider        string
	ProviderEventID string
	EventType       string
	PayloadSHA256   string
}
