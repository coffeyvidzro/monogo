package messaging

import (
	"encoding/json"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusSubmitted   Status = "submitted"
	StatusSent        Status = "sent"
	StatusDelivered   Status = "delivered"
	StatusUndelivered Status = "undelivered"
	StatusReceived    Status = "received"
	StatusFailed      Status = "failed"
)

type EventType string

const (
	EventQueued      EventType = "message.queued"
	EventSubmitted   EventType = "message.submitted"
	EventSent        EventType = "message.sent"
	EventDelivered   EventType = "message.delivered"
	EventUndelivered EventType = "message.undelivered"
	EventReceived    EventType = "message.received"
	EventFailed      EventType = "message.failed"
)

type Media struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
}

type CreateRequest struct {
	Channel Channel `json:"channel"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Body    *string `json:"body,omitempty"`
	Media   []Media `json:"media,omitempty"`
}

type InboundRequest struct {
	OrganizationID        uuid.UUID
	MessagingConnectionID uuid.UUID
	ProviderMessageID     string
	Channel               Channel
	From                  string
	To                    string
	Body                  *string
	Media                 []Media
}

type Failure struct {
	Code    *string
	Message *string
}

type ListRequest struct {
	Status    *string
	Direction *string
	Channel   *string
	Offset    int32
	Limit     int32
}

type Event struct {
	EventType      EventType       `json:"event_type"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	MessageID      uuid.UUID       `json:"message_id"`
	Resource       MessageResponse `json:"resource"`
	OccurredAt     time.Time       `json:"occurred_at"`
}

type MessageResponse struct {
	ID                    uuid.UUID       `json:"id"`
	OrganizationID        uuid.UUID       `json:"organization_id"`
	MessagingConnectionID *uuid.UUID      `json:"messaging_connection_id,omitempty"`
	Channel               string          `json:"channel"`
	Direction             string          `json:"direction"`
	Status                string          `json:"status"`
	From                  string          `json:"from"`
	To                    string          `json:"to"`
	Body                  *string         `json:"body,omitempty"`
	Media                 json.RawMessage `json:"media"`
	ProviderMessageID     *string         `json:"provider_message_id,omitempty"`
	FailureCode           *string         `json:"failure_code,omitempty"`
	FailureMessage        *string         `json:"failure_message,omitempty"`
	QueuedAt              *time.Time      `json:"queued_at,omitempty"`
	SubmittedAt           *time.Time      `json:"submitted_at,omitempty"`
	SentAt                *time.Time      `json:"sent_at,omitempty"`
	DeliveredAt           *time.Time      `json:"delivered_at,omitempty"`
	ReceivedAt            *time.Time      `json:"received_at,omitempty"`
	FailedAt              *time.Time      `json:"failed_at,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

func messageResponse(value sqlc.Message) MessageResponse {
	media := json.RawMessage(value.Media)
	if len(media) == 0 {
		media = json.RawMessage("[]")
	}
	return MessageResponse{
		ID:                    value.ID,
		OrganizationID:        value.OrganizationID,
		MessagingConnectionID: value.MessagingConnectionID,
		Channel:               value.Channel,
		Direction:             value.Direction,
		Status:                value.Status,
		From:                  value.FromAddress,
		To:                    value.ToAddress,
		Body:                  value.Body,
		Media:                 media,
		ProviderMessageID:     value.ProviderMessageID,
		FailureCode:           value.FailureCode,
		FailureMessage:        value.FailureMessage,
		QueuedAt:              pgconv.TimestamptzToTimePtr(value.QueuedAt),
		SubmittedAt:           pgconv.TimestamptzToTimePtr(value.SubmittedAt),
		SentAt:                pgconv.TimestamptzToTimePtr(value.SentAt),
		DeliveredAt:           pgconv.TimestamptzToTimePtr(value.DeliveredAt),
		ReceivedAt:            pgconv.TimestamptzToTimePtr(value.ReceivedAt),
		FailedAt:              pgconv.TimestamptzToTimePtr(value.FailedAt),
		CreatedAt:             pgconv.TimestamptzToTime(value.CreatedAt),
		UpdatedAt:             pgconv.TimestamptzToTime(value.UpdatedAt),
	}
}
