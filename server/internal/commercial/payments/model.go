package payments

import (
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
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

type CreateInput struct {
	Provider string `json:"provider"`
}

type Response struct {
	ID                  uuid.UUID  `json:"id"`
	CheckoutID          uuid.UUID  `json:"checkout_id"`
	Provider            string     `json:"provider"`
	ProviderReference   *string    `json:"provider_reference,omitempty"`
	AmountMinor         int64      `json:"amount_minor"`
	Currency            string     `json:"currency"`
	Status              string     `json:"status"`
	WalletTransactionID *uuid.UUID `json:"wallet_transaction_id,omitempty"`
	FailureCode         *string    `json:"failure_code,omitempty"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func response(row sqlc.Payment) Response {
	return Response{ID: row.ID, CheckoutID: row.CheckoutID, Provider: row.Provider,
		ProviderReference: row.ProviderReference, AmountMinor: row.AmountMinor, Currency: row.Currency,
		Status: row.Status, WalletTransactionID: row.WalletTransactionID, FailureCode: row.FailureCode,
		VerifiedAt: pgconv.TimestamptzToTimePtr(row.VerifiedAt), CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt)}
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

type RecoveryCandidate struct {
	OrganizationID    uuid.UUID
	PaymentID         uuid.UUID
	Provider          string
	ProviderReference string
	AmountMinor       int64
	Currency          string
}

type Verification struct {
	Succeeded   bool
	AmountMinor int64
	Currency    string
	VerifiedAt  time.Time
}
