package calling

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
)

type Consumer struct {
	service    *calls.Service
	controller calls.Controller
}

func NewConsumer(service *calls.Service, controller calls.Controller) *Consumer {
	if service == nil {
		panic("calling: calls service is required")
	}
	if controller == nil {
		panic("calling: calls controller is required")
	}
	return &Consumer{
		service:    service,
		controller: controller,
	}
}

func FreeSWITCHEvents() []string {
	return []string{
		"CHANNEL_CREATE",
		"CHANNEL_ANSWER",
		"CHANNEL_HOLD",
		"CHANNEL_UNHOLD",
		"CHANNEL_HANGUP_COMPLETE",
	}
}

func (c *Consumer) HandleFreeSWITCHEvent(
	ctx context.Context,
	event freeswitch.Event,
) error {
	if event.Name == "CHANNEL_CREATE" &&
		strings.TrimSpace(event.Header("variable_leamout_call_id")) == "" {
		admission, err := TranslateInboundFreeSWITCHEvent(event)
		switch {
		case errors.Is(err, ErrNotInboundAdmission):
		case err != nil:
			channelID := strings.TrimSpace(event.Header("Unique-ID"))
			if channelID != "" {
				if hangupErr := c.controller.Hangup(ctx, channelID); hangupErr != nil {
					return errors.Join(
						fmt.Errorf("translate inbound FreeSWITCH event: %w", err),
						fmt.Errorf("reject malformed inbound FreeSWITCH channel: %w", hangupErr),
					)
				}
			}
			return fmt.Errorf("translate inbound FreeSWITCH event: %w", err)
		default:
			if _, err := c.service.AdmitInbound(ctx, admission); err != nil {
				return fmt.Errorf("admit inbound call: %w", err)
			}
			return nil
		}
	}

	callEvent, err := TranslateFreeSWITCHEvent(event)
	switch {
	case errors.Is(err, ErrUnsupportedEvent),
		errors.Is(err, ErrUncorrelatedEvent):
		return nil
	case err != nil:
		return fmt.Errorf("translate call FreeSWITCH event: %w", err)
	}

	if err := c.service.ObserveLifecycle(ctx, callEvent); err != nil {
		return fmt.Errorf("handle call FreeSWITCH event: %w", err)
	}
	return nil
}
