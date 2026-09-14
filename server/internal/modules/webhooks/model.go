package webhooks

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
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

func endpointResponse(v sqlc.WebhookEndpoint) EndpointResponse {
	return EndpointResponse{v.ID, v.OrganizationID, v.Url, v.Enabled, v.SubscribedEvents, pgconv.TimestamptzToTime(v.CreatedAt), pgconv.TimestamptzToTime(v.UpdatedAt), pgconv.TimestamptzToTimePtr(v.DisabledAt), v.ConsecutiveFailures, pgconv.TimestamptzToTimePtr(v.LastFailureAt), v.DisabledReason}
}
func deliveryResponse(v sqlc.WebhookDelivery) DeliveryResponse {
	return DeliveryResponse{v.ID, v.EventID, v.EndpointID, v.Status, v.AttemptCount, v.ReplayCount, pgconv.TimestamptzToTime(v.NextAttemptAt), pgconv.TimestamptzToTimePtr(v.LastAttemptAt), pgconv.TimestamptzToTimePtr(v.LastReplayedAt), v.ResponseStatus, v.ResponseBody, v.LastError, pgconv.TimestamptzToTimePtr(v.DeliveredAt), pgconv.TimestamptzToTime(v.CreatedAt), pgconv.TimestamptzToTime(v.UpdatedAt)}
}
