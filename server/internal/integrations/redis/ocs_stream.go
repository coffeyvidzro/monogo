package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

type OCSEvent struct {
	StreamID               string
	OrganizationID         uuid.UUID
	WalletID               uuid.UUID
	ChargeID               *uuid.UUID
	OperationID            uuid.UUID
	WalletVersion          int64
	ChargeSequence         int64
	EventType              string
	BalanceDeltaMicros     int64
	ReservedDeltaMicros    int64
	BalanceAfterMicros     int64
	ReservedAfterMicros    int64
	ChargeAuthorizedMicros int64
	ChargeConsumedMicros   int64
	ChargeReservedMicros   int64
	ChargeStatus           string
	OccurredAt             time.Time
}

func (o *OCS) EnsureConsumerGroup(ctx context.Context, group string) error {
	if err := o.validateContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(group) == "" {
		return fmt.Errorf("OCS consumer group is required")
	}

	err := o.client.XGroupCreateMkStream(
		ctx,
		ocsEventStream,
		group,
		"0",
	).Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create OCS consumer group: %w", err)
	}

	return nil
}

func (o *OCS) ReadGroup(
	ctx context.Context,
	group string,
	consumer string,
	count int64,
	block time.Duration,
) ([]OCSEvent, error) {
	if err := validateStreamRead(o, ctx, group, consumer, count); err != nil {
		return nil, err
	}

	streams, err := o.client.XReadGroup(
		ctx,
		&redisv9.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams: []string{
				ocsEventStream,
				">",
			},
			Count: count,
			Block: block,
		},
	).Result()
	if errors.Is(err, redisv9.Nil) {
		return []OCSEvent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read OCS event stream: %w", err)
	}

	return decodeOCSStreams(streams)
}

func (o *OCS) RecoverPending(
	ctx context.Context,
	group string,
	consumer string,
	minimumIdle time.Duration,
	start string,
	count int64,
) ([]OCSEvent, string, error) {
	if err := validateStreamRead(o, ctx, group, consumer, count); err != nil {
		return nil, "", err
	}
	if minimumIdle < 0 {
		return nil, "", fmt.Errorf("OCS minimum idle time cannot be negative")
	}
	if strings.TrimSpace(start) == "" {
		start = "0-0"
	}

	messages, next, err := o.client.XAutoClaim(
		ctx,
		&redisv9.XAutoClaimArgs{
			Stream:   ocsEventStream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  minimumIdle,
			Start:    start,
			Count:    count,
		},
	).Result()
	if errors.Is(err, redisv9.Nil) {
		return []OCSEvent{}, "0-0", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("recover pending OCS events: %w", err)
	}

	events, err := decodeOCSMessages(messages)
	if err != nil {
		return nil, "", err
	}

	return events, next, nil
}

func (o *OCS) Acknowledge(
	ctx context.Context,
	group string,
	streamID string,
) error {
	if err := o.validateContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(group) == "" || strings.TrimSpace(streamID) == "" {
		return fmt.Errorf("OCS group and stream id are required")
	}

	acknowledged, err := o.client.XAck(
		ctx,
		ocsEventStream,
		group,
		streamID,
	).Result()
	if err != nil {
		return fmt.Errorf("acknowledge OCS event: %w", err)
	}
	if acknowledged != 1 {
		return fmt.Errorf("acknowledge OCS event %s: entry was not pending", streamID)
	}

	return nil
}

func validateStreamRead(
	o *OCS,
	ctx context.Context,
	group string,
	consumer string,
	count int64,
) error {
	if err := o.validateContext(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(group) == "" || strings.TrimSpace(consumer) == "" {
		return fmt.Errorf("OCS group and consumer are required")
	}
	if count <= 0 {
		return fmt.Errorf("OCS read count must be positive")
	}

	return nil
}

func decodeOCSStreams(streams []redisv9.XStream) ([]OCSEvent, error) {
	events := make([]OCSEvent, 0)
	for _, stream := range streams {
		decoded, err := decodeOCSMessages(stream.Messages)
		if err != nil {
			return nil, err
		}
		events = append(events, decoded...)
	}

	return events, nil
}

func decodeOCSMessages(messages []redisv9.XMessage) ([]OCSEvent, error) {
	events := make([]OCSEvent, 0, len(messages))
	for _, message := range messages {
		event, err := decodeOCSEvent(message)
		if err != nil {
			return nil, fmt.Errorf("decode OCS event %s: %w", message.ID, err)
		}
		events = append(events, event)
	}

	return events, nil
}

func decodeOCSEvent(message redisv9.XMessage) (OCSEvent, error) {
	organizationID, err := eventUUID(message.Values, "organization_id")
	if err != nil {
		return OCSEvent{}, err
	}
	walletID, err := eventUUID(message.Values, "wallet_id")
	if err != nil {
		return OCSEvent{}, err
	}
	operationID, err := eventUUID(message.Values, "operation_id")
	if err != nil {
		return OCSEvent{}, err
	}

	chargeID, err := optionalEventUUID(message.Values, "charge_id")
	if err != nil {
		return OCSEvent{}, err
	}

	walletVersion, err := eventInt64(message.Values, "wallet_version")
	if err != nil {
		return OCSEvent{}, err
	}
	chargeSequence, err := eventInt64(message.Values, "charge_sequence")
	if err != nil {
		return OCSEvent{}, err
	}
	balanceDelta, err := eventInt64(message.Values, "balance_delta_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	reservedDelta, err := eventInt64(message.Values, "reserved_delta_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	balanceAfter, err := eventInt64(message.Values, "balance_after_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	reservedAfter, err := eventInt64(message.Values, "reserved_after_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	chargeAuthorized, err := eventInt64(message.Values, "charge_authorized_after_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	chargeConsumed, err := eventInt64(message.Values, "charge_consumed_after_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	chargeReserved, err := eventInt64(message.Values, "charge_reserved_after_micros")
	if err != nil {
		return OCSEvent{}, err
	}
	occurredAtMillis, err := eventInt64(message.Values, "occurred_at_millis")
	if err != nil {
		return OCSEvent{}, err
	}

	eventType := eventString(message.Values, "event_type")
	chargeStatus := eventString(message.Values, "charge_status")
	if err := validateDecodedEvent(eventType, chargeStatus, chargeID); err != nil {
		return OCSEvent{}, err
	}

	return OCSEvent{
		StreamID:               message.ID,
		OrganizationID:         organizationID,
		WalletID:               walletID,
		ChargeID:               chargeID,
		OperationID:            operationID,
		WalletVersion:          walletVersion,
		ChargeSequence:         chargeSequence,
		EventType:              eventType,
		BalanceDeltaMicros:     balanceDelta,
		ReservedDeltaMicros:    reservedDelta,
		BalanceAfterMicros:     balanceAfter,
		ReservedAfterMicros:    reservedAfter,
		ChargeAuthorizedMicros: chargeAuthorized,
		ChargeConsumedMicros:   chargeConsumed,
		ChargeReservedMicros:   chargeReserved,
		ChargeStatus:           chargeStatus,
		OccurredAt:             time.UnixMilli(occurredAtMillis).UTC(),
	}, nil
}

func validateDecodedEvent(eventType string, chargeStatus string, chargeID *uuid.UUID) error {
	switch eventType {
	case "credit":
		if chargeID != nil || chargeStatus != "" {
			return fmt.Errorf("credit event cannot reference a charge")
		}
	case "reserve", "consume", "release", "debit":
		if chargeID == nil || chargeStatus != "active" {
			return fmt.Errorf("%s event requires an active charge", eventType)
		}
	case "finalize":
		if chargeID == nil {
			return fmt.Errorf("finalize event requires a charge")
		}
		switch chargeStatus {
		case "completed", "failed", "cancelled":
		default:
			return fmt.Errorf("finalize event has invalid charge status")
		}
	default:
		return fmt.Errorf("unsupported event type %q", eventType)
	}

	return nil
}

func eventString(values map[string]any, field string) string {
	return strings.TrimSpace(fmt.Sprint(values[field]))
}

func eventUUID(values map[string]any, field string) (uuid.UUID, error) {
	value, err := uuid.Parse(eventString(values, field))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s is invalid: %w", field, err)
	}

	return value, nil
}

func optionalEventUUID(values map[string]any, field string) (*uuid.UUID, error) {
	if eventString(values, field) == "" {
		return nil, nil
	}

	value, err := eventUUID(values, field)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func eventInt64(values map[string]any, field string) (int64, error) {
	value, err := strconv.ParseInt(
		eventString(values, field),
		10,
		64,
	)
	if err != nil {
		return 0, fmt.Errorf("%s is invalid: %w", field, err)
	}

	return value, nil
}
