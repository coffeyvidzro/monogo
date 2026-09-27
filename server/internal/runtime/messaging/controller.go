// Package messaging connects the canonical messaging domain to channel adapters.
package messaging

import (
	"context"
	"fmt"

	domain "github.com/coffeyvidzro/monogo/internal/telecom/messaging"
)

type Controller struct {
	sms      Sender
	whatsapp Sender
}

func NewController(sms, whatsapp Sender) *Controller {
	return &Controller{sms: sms, whatsapp: whatsapp}
}

// Send implements the domain Dispatcher without exposing adapter state.
func (c *Controller) Send(ctx context.Context, route domain.Route, message domain.OutboundMessage) (domain.Submission, error) {
	request, err := Translate(route, message)
	if err != nil {
		return domain.Submission{}, err
	}
	var sender Sender
	switch message.Channel {
	case domain.ChannelSMS, domain.ChannelMMS:
		sender = c.sms
	case domain.ChannelWhatsApp:
		sender = c.whatsapp
	default:
		return domain.Submission{}, fmt.Errorf("unsupported messaging channel %q", message.Channel)
	}
	if sender == nil {
		return domain.Submission{}, fmt.Errorf("messaging channel %q is not configured", message.Channel)
	}
	result, err := sender.Send(ctx, request)
	if err != nil {
		return domain.Submission{}, err
	}
	return domain.Submission{ProviderMessageID: result.ExternalID}, nil
}
