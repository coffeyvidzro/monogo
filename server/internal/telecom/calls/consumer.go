package calls

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

const (
	freeSWITCHEventChannelCreate         = "CHANNEL_CREATE"
	freeSWITCHEventChannelAnswer         = "CHANNEL_ANSWER"
	freeSWITCHEventChannelHold           = "CHANNEL_HOLD"
	freeSWITCHEventChannelUnhold         = "CHANNEL_UNHOLD"
	freeSWITCHEventChannelHangupComplete = "CHANNEL_HANGUP_COMPLETE"
)

type callLifecycleService interface {
	EnsureInbound(context.Context, InboundCallEvent) error
	MarkInboundAnswered(context.Context, InboundCallEvent) error
	FinishInbound(context.Context, InboundCallEvent) error
	MarkMediaHeld(context.Context, string) error
	MarkMediaResumed(context.Context, string) error
}

type Consumer struct {
	service callLifecycleService
}

func NewConsumer(service *Service) *Consumer {
	if service == nil {
		panic("calls: service is required")
	}
	return &Consumer{service: service}
}

func (c *Consumer) HandleFreeSWITCHEvent(ctx context.Context, event freeswitch.Event) error {
	switch event.Name {
	case freeSWITCHEventChannelHold:
		return c.service.MarkMediaHeld(ctx, mediaChannelID(event))
	case freeSWITCHEventChannelUnhold:
		return c.service.MarkMediaResumed(ctx, mediaChannelID(event))
	case freeSWITCHEventChannelCreate, freeSWITCHEventChannelAnswer, freeSWITCHEventChannelHangupComplete:
	default:
		return nil
	}

	input, trusted, err := inboundCallEvent(event)
	if err != nil {
		return err
	}
	if !trusted {
		return nil
	}

	switch event.Name {
	case freeSWITCHEventChannelCreate:
		return c.service.EnsureInbound(ctx, input)
	case freeSWITCHEventChannelAnswer:
		return c.service.MarkInboundAnswered(ctx, input)
	case freeSWITCHEventChannelHangupComplete:
		return c.service.FinishInbound(ctx, input)
	default:
		return nil
	}
}

func mediaChannelID(event freeswitch.Event) string {
	return strings.TrimSpace(event.Header("Unique-ID"))
}

func inboundCallEvent(event freeswitch.Event) (InboundCallEvent, bool, error) {
	application := strings.TrimSpace(event.Header("variable_sip_h_X-Leamout-Voice-Application-ID"))
	if application == "" {
		return InboundCallEvent{}, false, nil
	}

	organizationID, err := uuid.Parse(strings.TrimSpace(event.Header("variable_sip_h_X-Leamout-Organization-ID")))
	if err != nil {
		return InboundCallEvent{}, true, fmt.Errorf("parse inbound organization id: %w", err)
	}
	applicationID, err := uuid.Parse(application)
	if err != nil {
		return InboundCallEvent{}, true, fmt.Errorf("parse inbound application id: %w", err)
	}
	carrierConnectionID, err := uuid.Parse(strings.TrimSpace(event.Header("variable_sip_h_X-Leamout-Carrier-Connection-ID")))
	if err != nil {
		return InboundCallEvent{}, true, fmt.Errorf("parse inbound carrier connection id: %w", err)
	}
	phoneNumberID, err := uuid.Parse(strings.TrimSpace(event.Header("variable_sip_h_X-Leamout-Phone-Number-ID")))
	if err != nil {
		return InboundCallEvent{}, true, fmt.Errorf("parse inbound phone number id: %w", err)
	}

	channelID := strings.TrimSpace(event.Header("Unique-ID"))
	if channelID == "" {
		return InboundCallEvent{}, true, fmt.Errorf("inbound FreeSWITCH event is missing Unique-ID")
	}

	from := firstNonEmpty(
		event.Header("Caller-Caller-ID-Number"),
		event.Header("variable_sip_from_user"),
	)
	to := firstNonEmpty(
		event.Header("Caller-Destination-Number"),
		event.Header("variable_sip_to_user"),
	)
	if from == "" {
		from = "anonymous"
	}
	if to == "" {
		return InboundCallEvent{}, true, fmt.Errorf("inbound FreeSWITCH event is missing destination")
	}

	return InboundCallEvent{
		OrganizationID:      organizationID,
		CarrierConnectionID: carrierConnectionID,
		PhoneNumberID:       phoneNumberID,
		ApplicationID:       applicationID,
		ChannelID:           channelID,
		From:                from,
		To:                  to,
		HangupCause:         strings.TrimSpace(event.Header("Hangup-Cause")),
		WasAnswered:         channelWasAnswered(event),
	}, true, nil
}

func channelWasAnswered(event freeswitch.Event) bool {
	answerEpoch := strings.TrimSpace(event.Header("variable_answer_epoch"))
	return answerEpoch != "" && answerEpoch != "0"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
