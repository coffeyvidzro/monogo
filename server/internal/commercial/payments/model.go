package payments

import (
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

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

// VerifiedSettlement is a successful charge result obtained from an
// authenticated webhook or a direct provider API verification. Callers must
// never construct it from customer-supplied status data.
type VerifiedSettlement struct {
	OrganizationID    uuid.UUID
	PaymentID         uuid.UUID
	Provider          string
	ProviderReference string
	AmountMinor       int64
	Currency          string
	VerifiedAt        time.Time
}

// SettlementResult is the durable, atomically linked result of a top-up.
type SettlementResult struct {
	Payment     sqlc.Payment
	Checkout    sqlc.Checkout
	Transaction sqlc.WalletTransaction
}
