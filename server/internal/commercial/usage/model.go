package usage

import (
	"encoding/json"
	"time"

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
