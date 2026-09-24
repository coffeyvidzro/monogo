package checkout

import (
	"time"

	"github.com/google/uuid"
)

// CreateRequest is an internal wallet top-up intent, not a payment or credit.
type CreateRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	AmountMinor    int64
	Currency       string
	IdempotencyKey string
	ExpiresAt      time.Time
}
