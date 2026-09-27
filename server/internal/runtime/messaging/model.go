package messaging

import (
	"github.com/google/uuid"
)

type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

type Request struct {
	MessageID, ConnectionID uuid.UUID
	Channel                 Channel
	From, To, Text          string
}
type Result struct{ ExternalID string }
