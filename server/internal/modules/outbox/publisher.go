package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leamout/leamout/internal/database/sqlc"
	natsintegration "github.com/leamout/leamout/internal/integrations/nats"
)

const (
	headerEventID       = "X-Leamout-Event-ID"
	headerAggregateType = "X-Leamout-Aggregate-Type"
	headerAggregateID   = "X-Leamout-Aggregate-ID"
)

type messagePublisher interface {
	PublishWithOptions(context.Context, string, []byte, map[string]string, string) error
}

type Publisher struct {
	client messagePublisher
}

func NewPublisher(client messagePublisher) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) Publish(ctx context.Context, event sqlc.OutboxEvent) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("outbox publisher client is required")
	}

	subject, err := eventSubject(event.Subject)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	if len(event.Headers) > 0 {
		if err := json.Unmarshal(event.Headers, &headers); err != nil {
			return fmt.Errorf("decode outbox event %s headers: %w", event.ID, err)
		}
	}
	headers[headerEventID] = event.ID.String()
	headers[headerAggregateType] = event.AggregateType
	headers[headerAggregateID] = event.AggregateID.String()

	if err := p.client.PublishWithOptions(ctx, subject, event.Payload, headers, event.ID.String()); err != nil {
		return fmt.Errorf("publish outbox event %s: %w", event.ID, err)
	}
	return nil
}

func eventSubject(subject string) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", fmt.Errorf("outbox event subject is required")
	}
	if strings.ContainsAny(subject, " \t\r\n*> ") {
		return "", fmt.Errorf("invalid outbox event subject %q", subject)
	}
	if strings.HasPrefix(subject, natsintegration.EventsSubjectPrefix) {
		return subject, nil
	}
	return natsintegration.EventsSubjectPrefix + subject, nil
}
