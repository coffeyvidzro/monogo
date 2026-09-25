package checkout

import (
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type CreateInput struct {
	WalletID    uuid.UUID `json:"wallet_id"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CreateRequest is an internal wallet top-up intent, not a payment or credit.
type CreateRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	AmountMinor    int64
	Currency       string
	IdempotencyKey string
	ExpiresAt      time.Time
}

type Response struct {
	ID                    uuid.UUID  `json:"id"`
	WalletID              uuid.UUID  `json:"wallet_id"`
	AmountMinor           int64      `json:"amount_minor"`
	Currency              string     `json:"currency"`
	Status                string     `json:"status"`
	CreditedTransactionID *uuid.UUID `json:"credited_transaction_id,omitempty"`
	ExpiresAt             time.Time  `json:"expires_at"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func response(row sqlc.Checkout) Response {
	return Response{ID: row.ID, WalletID: row.WalletID, AmountMinor: row.AmountMinor,
		Currency: row.Currency, Status: row.Status, CreditedTransactionID: row.CreditedTransactionID,
		ExpiresAt: pgconv.TimestamptzToTime(row.ExpiresAt), CompletedAt: pgconv.TimestamptzToTimePtr(row.CompletedAt),
		CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt)}
}
