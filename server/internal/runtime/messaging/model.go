package messaging

import (
	"context"

	"github.com/google/uuid"
)

// Request is the runtime boundary shared by the two channel adapters.
type Request struct {
	MessageID    uuid.UUID
	ConnectionID uuid.UUID
	From         string
	To           string
	Text         string
}

type Result struct{ ExternalID string }

type Sender interface {
	Send(context.Context, Request) (Result, error)
}
