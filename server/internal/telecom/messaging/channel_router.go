package messaging

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// OutboundMessage is the canonical command shared by all channel transports.
// MessageID is stable across retries and is the adapter's idempotency key.
type OutboundMessage struct {
	MessageID uuid.UUID
	Channel   Channel
	From      string
	To        string
	Body      *string
	Media     []Media
}

// Route identifies the configured connection selected by domain routing. The
// adapter resolves credentials and all protocol-specific details internally.
type Route struct {
	CarrierConnectionID uuid.UUID
}

// Submission is the only transport response retained by the canonical domain.
// Raw SMPP, Meta/BSP, and RCS provider state stays inside its adapter.
type Submission struct {
	ProviderMessageID string
}

// Transport translates a canonical message to one channel protocol.
type Transport interface {
	Send(context.Context, Route, OutboundMessage) (Submission, error)
}

// ChannelTransports are the three supported transport families. SMS also owns
// MMS because both are delivered over the carrier messaging connection.
type ChannelTransports struct {
	SMS      Transport // direct carrier SMPP
	WhatsApp Transport // Meta or BSP API
	RCS      Transport // configured RCS provider
}

// ChannelRouter chooses only by canonical channel; carrier selection remains a
// separate domain-routing concern.
type ChannelRouter struct {
	transports ChannelTransports
}

func NewChannelRouter(transports ChannelTransports) *ChannelRouter {
	return &ChannelRouter{transports: transports}
}

func (r *ChannelRouter) Transport(channel Channel) (Transport, error) {
	if r == nil {
		return nil, fmt.Errorf("messaging channel router is required")
	}
	var transport Transport
	switch channel {
	case ChannelSMS, ChannelMMS:
		transport = r.transports.SMS
	case ChannelWhatsApp:
		transport = r.transports.WhatsApp
	case ChannelRCS:
		transport = r.transports.RCS
	default:
		return nil, fmt.Errorf("unsupported messaging channel %q", channel)
	}
	if transport == nil {
		return nil, fmt.Errorf("messaging channel %q is not configured", channel)
	}
	return transport, nil
}
