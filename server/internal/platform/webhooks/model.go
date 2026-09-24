package webhooks

import (
	"encoding/json"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type CreateRequest struct {
	URL              string   `json:"url"`
	SubscribedEvents []string `json:"subscribed_events"`
	Enabled          *bool    `json:"enabled,omitempty"`
}
type UpdateRequest struct {
	URL              *string   `json:"url,omitempty"`
	SubscribedEvents *[]string `json:"subscribed_events,omitempty"`
	Enabled          *bool     `json:"enabled,omitempty"`
}

type InboundEvent struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	EventType      string
	ObjectType     string
	ObjectID       *uuid.UUID
	Payload        json.RawMessage
	OccurredAt     time.Time
}

type DeliveryEnvelope struct {
	ID         uuid.UUID       `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

type DeliveryAttempt struct {
	StatusCode *int32
	Body       *string
	Err        error
}

type EndpointResponse struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	URL                 string     `json:"url"`
	Enabled             bool       `json:"enabled"`
	SubscribedEvents    []string   `json:"subscribed_events"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DisabledAt          *time.Time `json:"disabled_at,omitempty"`
	ConsecutiveFailures int32      `json:"consecutive_failures"`
	LastFailureAt       *time.Time `json:"last_failure_at,omitempty"`
	DisabledReason      *string    `json:"disabled_reason,omitempty"`
}
type SecretResponse struct {
	SigningSecret string `json:"signing_secret"`
}
type DeliveryResponse struct {
	ID             uuid.UUID  `json:"id"`
	EventID        uuid.UUID  `json:"event_id"`
	EndpointID     uuid.UUID  `json:"endpoint_id"`
	Status         string     `json:"status"`
	AttemptCount   int32      `json:"attempt_count"`
	ReplayCount    int32      `json:"replay_count"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty"`
	LastReplayedAt *time.Time `json:"last_replayed_at,omitempty"`
	ResponseStatus *int32     `json:"response_status,omitempty"`
	ResponseBody   *string    `json:"response_body,omitempty"`
	LastError      *string    `json:"last_error,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func endpointResponse(webhook sqlc.WebhookEndpoint) EndpointResponse {
	return EndpointResponse{
		ID:                  webhook.ID,
		OrganizationID:      webhook.OrganizationID,
		URL:                 webhook.Url,
		Enabled:             webhook.Enabled,
		SubscribedEvents:    webhook.SubscribedEvents,
		CreatedAt:           pgconv.TimestamptzToTime(webhook.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(webhook.UpdatedAt),
		DisabledAt:          pgconv.TimestamptzToTimePtr(webhook.DisabledAt),
		ConsecutiveFailures: webhook.ConsecutiveFailures,
		LastFailureAt:       pgconv.TimestamptzToTimePtr(webhook.LastFailureAt),
		DisabledReason:      webhook.DisabledReason,
	}
}

func deliveryResponse(delivery sqlc.WebhookDelivery) DeliveryResponse {
	return DeliveryResponse{
		ID:             delivery.ID,
		EventID:        delivery.EventID,
		EndpointID:     delivery.EndpointID,
		Status:         delivery.Status,
		AttemptCount:   delivery.AttemptCount,
		ReplayCount:    delivery.ReplayCount,
		NextAttemptAt:  pgconv.TimestamptzToTime(delivery.NextAttemptAt),
		LastAttemptAt:  pgconv.TimestamptzToTimePtr(delivery.LastAttemptAt),
		LastReplayedAt: pgconv.TimestamptzToTimePtr(delivery.LastReplayedAt),
		ResponseStatus: delivery.ResponseStatus,
		ResponseBody:   delivery.ResponseBody,
		LastError:      delivery.LastError,
		DeliveredAt:    pgconv.TimestamptzToTimePtr(delivery.DeliveredAt),
		CreatedAt:      pgconv.TimestamptzToTime(delivery.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(delivery.UpdatedAt),
	}
}
