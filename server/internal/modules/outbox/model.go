package outbox

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Subject       string
	AggregateType string
	AggregateID   uuid.UUID
	Payload       any
	Headers       map[string]string
	AvailableAt   *time.Time
}
