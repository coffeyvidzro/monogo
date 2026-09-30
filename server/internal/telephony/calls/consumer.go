package calls

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
)

type Consumer struct {
	service *Service
}

func NewConsumer(service *Service) *Consumer {
	if service == nil {
		panic("calls: service is required")
	}
	return &Consumer{service: service}
}

func (c *Consumer) HandleInbound(
	ctx context.Context,
	event calling.InboundEvent,
) error {
	_, err := c.service.AdmitInbound(ctx, InboundAdmissionRequest{
		ChannelID:           event.ChannelID,
		SIPCallID:           event.SIPCallID,
		OrganizationID:      event.OrganizationID,
		ApplicationID:       event.ApplicationID,
		PhoneNumberID:       event.PhoneNumberID,
		VoiceBindingID:      event.VoiceBindingID,
		CarrierConnectionID: event.CarrierConnectionID,
		FromURI:             event.FromURI,
		ToURI:               event.ToURI,
		OccurredAt:          event.OccurredAt,
	})
	if err != nil {
		return fmt.Errorf("admit inbound call: %w", err)
	}
	return nil
}

func (c *Consumer) HandleLifecycle(
	ctx context.Context,
	event calling.LifecycleEvent,
) error {
	if err := c.service.ObserveLifecycle(ctx, LifecycleEvent{
		CallID:       event.CallID,
		ChannelID:    event.ChannelID,
		SIPCallID:    event.SIPCallID,
		Type:         LifecycleEventType(event.Type),
		OccurredAt:   event.OccurredAt,
		HangupReason: event.HangupReason,
	}); err != nil {
		return fmt.Errorf("handle call lifecycle event: %w", err)
	}
	return nil
}
