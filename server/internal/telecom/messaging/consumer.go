package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

// OutboundMessage is the provider-neutral command passed to a carrier adapter.
// MessageID is stable across retries and must be used as the provider request's
// idempotency key when the provider supports one.
type OutboundMessage struct {
	MessageID uuid.UUID
	Channel   Channel
	From      string
	To        string
	Body      *string
	Media     []Media
}

// Submission is the normalized result of a carrier accepting a message.
type Submission struct {
	ProviderMessageID string
}

// Provider is implemented by each SMS/MMS carrier adapter. Provider-specific
// request and response shapes must not escape the adapter.
type Provider interface {
	Send(context.Context, OutboundMessage) (Submission, error)
}

// ProviderSelection binds an outbound message to the selected carrier adapter.
type ProviderSelection struct {
	CarrierConnectionID uuid.UUID
	Provider            Provider
}

// ProviderResolver applies routing policy without coupling messaging to a
// particular carrier SDK.
type ProviderResolver interface {
	Resolve(context.Context, sqlc.Message) (ProviderSelection, error)
}

type messageLifecycle interface {
	SetProviderAttribution(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (sqlc.Message, error)
	MarkSubmitted(context.Context, uuid.UUID, uuid.UUID, string) (sqlc.Message, error)
	MarkSent(context.Context, uuid.UUID, uuid.UUID) (sqlc.Message, error)
	MarkDelivered(context.Context, uuid.UUID, uuid.UUID) (sqlc.Message, error)
	MarkUndelivered(context.Context, uuid.UUID, uuid.UUID, Failure) (sqlc.Message, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, Failure) (sqlc.Message, error)
	CreateInbound(context.Context, InboundRequest) (sqlc.Message, error)
	GetByProviderID(context.Context, uuid.UUID, string) (sqlc.Message, error)
}

// Consumer translates between the durable message lifecycle and carrier
// adapters. It intentionally contains no provider-specific parsing.
type Consumer struct {
	messages messageLifecycle
	resolver ProviderResolver
}

func NewConsumer(messages messageLifecycle, resolver ProviderResolver) *Consumer {
	if messages == nil || resolver == nil {
		panic("messaging: lifecycle and provider resolver are required")
	}
	return &Consumer{messages: messages, resolver: resolver}
}

// HandleQueued submits a queued outbound message. Infrastructure errors are
// returned so the caller can retry with the same stable MessageID.
func (c *Consumer) HandleQueued(ctx context.Context, message sqlc.Message) (sqlc.Message, error) {
	if message.Direction != string(DirectionOutbound) || message.Status != string(StatusQueued) {
		return sqlc.Message{}, fmt.Errorf("message %s is not queued outbound", message.ID)
	}
	selection, err := c.resolver.Resolve(ctx, message)
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("resolve messaging provider: %w", err)
	}
	if selection.CarrierConnectionID == uuid.Nil || selection.Provider == nil {
		return sqlc.Message{}, fmt.Errorf("resolve messaging provider: invalid selection")
	}
	attributed, err := c.messages.SetProviderAttribution(ctx, message.OrganizationID, message.ID, selection.CarrierConnectionID)
	if err != nil {
		return sqlc.Message{}, err
	}

	var media []Media
	if err := unmarshalMedia(attributed.Media, &media); err != nil {
		return sqlc.Message{}, err
	}
	submission, err := selection.Provider.Send(ctx, OutboundMessage{
		MessageID: attributed.ID, Channel: Channel(attributed.Channel),
		From: attributed.FromAddress, To: attributed.ToAddress,
		Body: attributed.Body, Media: media,
	})
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("submit message to provider: %w", err)
	}
	providerID := strings.TrimSpace(submission.ProviderMessageID)
	if providerID == "" {
		code, detail := "invalid_provider_response", "provider accepted message without an id"
		return c.messages.MarkFailed(ctx, message.OrganizationID, message.ID, Failure{Code: &code, Message: &detail})
	}
	return c.messages.MarkSubmitted(ctx, message.OrganizationID, message.ID, providerID)
}

func unmarshalMedia(payload []byte, target *[]Media) error {
	if len(payload) == 0 {
		*target = []Media{}
		return nil
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode message media: %w", err)
	}
	return nil
}
