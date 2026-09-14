package nats

import (
	"context"
	"fmt"
	"strings"

	natsjs "github.com/nats-io/nats.go/jetstream"
)

type AckAction uint8

const (
	AckNone AckAction = iota
	Ack
	Nak
	Term
	InProgress
)

type ConsumerHandler func(context.Context, natsjs.Msg) (AckAction, error)

func (c *Client) CreateOrUpdateConsumer(ctx context.Context, stream string, config natsjs.ConsumerConfig) (natsjs.Consumer, error) {
	if err := c.validateConsumer(ctx, stream, config); err != nil {
		return nil, err
	}

	consumer, err := c.jetStream.CreateOrUpdateConsumer(ctx, stream, config)
	if err != nil {
		return nil, fmt.Errorf("create or update NATS consumer on %q: %w", stream, err)
	}

	return consumer, nil
}

// Consume starts a pull consumer and blocks until ctx is cancelled.
// The handler chooses exactly one acknowledgement action for every message.
// Returning AckNone is invalid and does not acknowledge the message.
func (c *Client) Consume(ctx context.Context, consumer natsjs.Consumer, handler ConsumerHandler) error {
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("NATS context is required")
	}
	if consumer == nil {
		return fmt.Errorf("NATS consumer is required")
	}
	if handler == nil {
		return fmt.Errorf("NATS consumer handler is required")
	}

	cons, err := consumer.Consume(func(msg natsjs.Msg) {
		action, handlerErr := handler(ctx, msg)
		if err := applyAckAction(msg, action); err != nil {
			return
		}
		_ = handlerErr
	})
	if err != nil {
		return fmt.Errorf("start NATS consumer: %w", err)
	}
	defer cons.Stop()

	<-ctx.Done()
	return ctx.Err()
}

func applyAckAction(msg natsjs.Msg, action AckAction) error {
	var err error

	switch action {
	case Ack:
		err = msg.Ack()
	case Nak:
		err = msg.Nak()
	case Term:
		err = msg.Term()
	case InProgress:
		err = msg.InProgress()
	case AckNone:
		return fmt.Errorf("NATS acknowledgement action is required")
	default:
		return fmt.Errorf("invalid NATS acknowledgement action: %d", action)
	}

	if err != nil {
		return fmt.Errorf("apply NATS acknowledgement action %d: %w", action, err)
	}
	return nil
}

func (c *Client) validateConsumer(ctx context.Context, stream string, config natsjs.ConsumerConfig) error {
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("NATS context is required")
	}
	if stream = strings.TrimSpace(stream); stream == "" {
		return fmt.Errorf("NATS stream name is required")
	}
	if strings.TrimSpace(config.Name) == "" && strings.TrimSpace(config.Durable) == "" {
		return fmt.Errorf("NATS consumer name or durable name is required")
	}

	return nil
}
