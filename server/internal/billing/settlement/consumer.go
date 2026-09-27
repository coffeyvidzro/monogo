package settlement

import (
	"context"
	"fmt"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

type ConsumerConfig struct {
	Group       string
	Consumer    string
	BatchSize   int64
	Block       time.Duration
	MinimumIdle time.Duration
}

func DefaultConsumerConfig(consumer string) ConsumerConfig {
	return ConsumerConfig{
		Group:       "billing-ocs-persistence",
		Consumer:    strings.TrimSpace(consumer),
		BatchSize:   100,
		Block:       2 * time.Second,
		MinimumIdle: 30 * time.Second,
	}
}

type Consumer struct {
	ocs       *redisintegration.OCS
	persister *Persister
	config    ConsumerConfig
	logger    *logging.Logger
}

func NewConsumer(
	ocs *redisintegration.OCS,
	persister *Persister,
	config ConsumerConfig,
	logger *logging.Logger,
) (*Consumer, error) {
	if ocs == nil {
		return nil, fmt.Errorf("billing settlement: OCS is required")
	}
	if persister == nil {
		return nil, fmt.Errorf("billing settlement: persister is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("billing settlement: logger is required")
	}
	if strings.TrimSpace(config.Group) == "" || strings.TrimSpace(config.Consumer) == "" {
		return nil, fmt.Errorf("billing settlement: group and consumer are required")
	}
	if config.BatchSize <= 0 || config.Block <= 0 || config.MinimumIdle < 0 {
		return nil, fmt.Errorf("billing settlement: invalid consumer timing or batch configuration")
	}

	return &Consumer{
		ocs:       ocs,
		persister: persister,
		config:    config,
		logger:    logger,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ocs.EnsureConsumerGroup(ctx, c.config.Group); err != nil {
		return err
	}
	backoff := 250 * time.Millisecond
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := c.recoverPending(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error(
				ctx,
				"recover pending OCS events",
				"error", err,
			)
			if !waitForRetry(ctx, backoff) {
				return nil
			}
			backoff = nextBackoff(backoff)
			continue
		}

		events, err := c.ocs.ReadGroup(
			ctx,
			c.config.Group,
			c.config.Consumer,
			c.config.BatchSize,
			c.config.Block,
		)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error(
				ctx,
				"read OCS events",
				"error", err,
			)
			if !waitForRetry(ctx, backoff) {
				return nil
			}
			backoff = nextBackoff(backoff)
			continue
		}
		if err := c.process(ctx, events); err != nil {
			retryDelay := backoff
			if retryDelay < c.config.MinimumIdle {
				retryDelay = c.config.MinimumIdle
			}
			if !waitForRetry(ctx, retryDelay) {
				return nil
			}
			backoff = nextBackoff(backoff)
			continue
		}
		backoff = 250 * time.Millisecond
	}
}

func (c *Consumer) recoverPending(ctx context.Context) error {
	start := "0-0"
	for {
		events, next, err := c.ocs.RecoverPending(
			ctx,
			c.config.Group,
			c.config.Consumer,
			c.config.MinimumIdle,
			start,
			c.config.BatchSize,
		)
		if err != nil {
			return err
		}
		if err := c.process(ctx, events); err != nil {
			return err
		}
		if next == "0-0" {
			return nil
		}
		start = next
	}
}

func (c *Consumer) process(ctx context.Context, events []redisintegration.OCSEvent) error {
	for _, event := range events {
		result, err := c.persister.Persist(ctx, event)
		if err != nil {
			c.logFailure(ctx, event, result.Outcome, err)
			return fmt.Errorf("persist OCS event %s: %w", event.StreamID, err)
		}
		if err := c.ocs.Acknowledge(ctx, c.config.Group, event.StreamID); err != nil {
			return err
		}
	}

	return nil
}

func (c *Consumer) logFailure(
	ctx context.Context,
	event redisintegration.OCSEvent,
	outcome Outcome,
	err error,
) {
	chargeID := ""
	if event.ChargeID != nil {
		chargeID = event.ChargeID.String()
	}
	c.logger.Error(
		ctx,
		"OCS financial event persistence failed",
		"outcome", outcome,
		"stream_id", event.StreamID,
		"operation_id", event.OperationID,
		"organization_id", event.OrganizationID,
		"wallet_id", event.WalletID,
		"charge_id", chargeID,
		"wallet_version", event.WalletVersion,
		"charge_sequence", event.ChargeSequence,
		"event_type", event.EventType,
		"error", err,
	)
}

func waitForRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > 5*time.Second {
		return 5 * time.Second
	}

	return next
}
