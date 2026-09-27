package settlement

import (
	"context"
	"fmt"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
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
}

func NewConsumer(
	ocs *redisintegration.OCS,
	persister *Persister,
	config ConsumerConfig,
) (*Consumer, error) {
	if ocs == nil {
		return nil, fmt.Errorf("billing settlement: OCS is required")
	}
	if persister == nil {
		return nil, fmt.Errorf("billing settlement: persister is required")
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
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ocs.EnsureConsumerGroup(ctx, c.config.Group); err != nil {
		return err
	}
	if err := c.recoverPending(ctx); err != nil {
		return err
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := c.recoverPending(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
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
			return err
		}
		if err := c.process(ctx, events); err != nil {
			return err
		}
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
		if err := c.persister.Persist(ctx, event); err != nil {
			return fmt.Errorf("persist OCS event %s: %w", event.StreamID, err)
		}
		if err := c.ocs.Acknowledge(ctx, c.config.Group, event.StreamID); err != nil {
			return err
		}
	}

	return nil
}
