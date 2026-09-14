package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	natsjs "github.com/nats-io/nats.go/jetstream"

	natsintegration "github.com/leamout/leamout/internal/integrations/nats"
)

const webhookConsumerName = "webhooks"

type Consumer struct {
	client  *natsintegration.Client
	service *Service
}

func NewConsumer(client *natsintegration.Client, service *Service) *Consumer {
	return &Consumer{client: client, service: service}
}

func (c *Consumer) Run(ctx context.Context) error {
	consumer, err := c.client.CreateOrUpdateConsumer(ctx, natsintegration.EventsStreamName, natsjs.ConsumerConfig{
		Name:          webhookConsumerName,
		Durable:       webhookConsumerName,
		FilterSubject: natsintegration.EventsSubject,
		AckPolicy:     natsjs.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    20,
	})
	if err != nil {
		return fmt.Errorf("initialize webhook NATS consumer: %w", err)
	}

	err = c.client.Consume(ctx, consumer, func(messageCtx context.Context, message natsjs.Msg) (natsintegration.AckAction, error) {
		event, err := inboundEventFromMessage(message)
		if err != nil {
			return natsintegration.Term, err
		}
		if err := c.service.Ingest(messageCtx, event); err != nil {
			return natsintegration.Nak, err
		}
		return natsintegration.Ack, nil
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func inboundEventFromMessage(message natsjs.Msg) (InboundEvent, error) {
	headers := message.Headers()
	eventID, err := uuid.Parse(strings.TrimSpace(headers.Get("X-Leamout-Event-ID")))
	if err != nil {
		return InboundEvent{}, fmt.Errorf("invalid webhook event id: %w", err)
	}
	organizationID, err := uuid.Parse(strings.TrimSpace(headers.Get("organization_id")))
	if err != nil {
		return InboundEvent{}, fmt.Errorf("invalid webhook organization id: %w", err)
	}

	eventType := strings.TrimSpace(headers.Get("event_type"))
	if eventType == "" {
		eventType = strings.TrimPrefix(message.Subject(), natsintegration.EventsSubjectPrefix)
	}
	objectType := strings.TrimSpace(headers.Get("X-Leamout-Aggregate-Type"))
	if eventType == "" || objectType == "" {
		return InboundEvent{}, fmt.Errorf("webhook event metadata is incomplete")
	}

	var objectID *uuid.UUID
	if value := strings.TrimSpace(headers.Get("X-Leamout-Aggregate-ID")); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return InboundEvent{}, fmt.Errorf("invalid webhook object id: %w", err)
		}
		objectID = &parsed
	}

	payload := append([]byte(nil), message.Data()...)
	var metadata struct {
		OccurredAt time.Time `json:"occurred_at"`
	}
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return InboundEvent{}, fmt.Errorf("decode webhook event payload: %w", err)
	}

	return InboundEvent{
		ID:             eventID,
		OrganizationID: organizationID,
		EventType:      eventType,
		ObjectType:     objectType,
		ObjectID:       objectID,
		Payload:        payload,
		OccurredAt:     metadata.OccurredAt,
	}, nil
}
