package usage

import (
	"encoding/json"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

// RecordRequest is an internal, non-financial usage observation. The
// idempotency key must be stable for retries of the same source observation.
type RecordRequest struct {
	OrganizationID uuid.UUID
	MeterID        uuid.UUID
	Quantity       int64
	SourceType     string
	SourceID       string
	IdempotencyKey string
	Dimensions     json.RawMessage
	OccurredAt     time.Time
}

// ChargeRequest is produced by a trusted rating component. AmountMinor is the
// final rated amount, not a customer-controlled unit price.
type ChargeRequest struct {
	Usage       RecordRequest
	Currency    string
	AmountMinor int64
}

type ChargeResult struct {
	UsageEvent  sqlc.UsageEvent
	Charge      sqlc.UsageCharge
	Transaction sqlc.WalletTransaction
}
