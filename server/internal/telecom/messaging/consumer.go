package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

// RouteResolver applies tenant routing policy and selects a connection. It
// deliberately does not select or expose a protocol adapter.
type RouteResolver interface {
	Resolve(context.Context, sqlc.Message) (Route, error)
}

type outboundLifecycle interface {
	SetProviderAttribution(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (sqlc.Message, error)
	MarkSubmitted(context.Context, uuid.UUID, uuid.UUID, string) (sqlc.Message, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, Failure) (sqlc.Message, error)
}

// Consumer coordinates domain routing, channel routing, and lifecycle changes.
type Consumer struct {
	messages outboundLifecycle
	routes   RouteResolver
	channels *ChannelRouter
}

func NewConsumer(messages outboundLifecycle, routes RouteResolver, channels *ChannelRouter) *Consumer {
	if messages == nil || routes == nil || channels == nil {
		panic("messaging: lifecycle, route resolver, and channel router are required")
	}
	return &Consumer{messages: messages, routes: routes, channels: channels}
}

// HandleQueued submits a queued outbound message. Infrastructure errors are
// returned so the caller can retry with the same stable MessageID.
func (c *Consumer) HandleQueued(ctx context.Context, message sqlc.Message) (sqlc.Message, error) {
	if message.Direction != string(DirectionOutbound) || message.Status != string(StatusQueued) {
		return sqlc.Message{}, fmt.Errorf("message %s is not queued outbound", message.ID)
	}
	channel := Channel(message.Channel)
	transport, err := c.channels.Transport(channel)
	if err != nil {
		return sqlc.Message{}, err
	}
	route, err := c.routes.Resolve(ctx, message)
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("resolve message route: %w", err)
	}
	if route.CarrierConnectionID == uuid.Nil {
		return sqlc.Message{}, fmt.Errorf("resolve message route: carrier connection is required")
	}
	attributed, err := c.messages.SetProviderAttribution(ctx, message.OrganizationID, message.ID, route.CarrierConnectionID)
	if err != nil {
		return sqlc.Message{}, err
	}

	var media []Media
	if err := unmarshalMedia(attributed.Media, &media); err != nil {
		return sqlc.Message{}, err
	}
	submission, err := transport.Send(ctx, route, OutboundMessage{
		MessageID: attributed.ID, Channel: Channel(attributed.Channel),
		From: attributed.FromAddress, To: attributed.ToAddress,
		Body: attributed.Body, Media: media,
	})
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("submit %s message: %w", channel, err)
	}
	providerID := strings.TrimSpace(submission.ProviderMessageID)
	if providerID == "" {
		code, detail := "invalid_provider_response", "transport accepted message without an id"
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
