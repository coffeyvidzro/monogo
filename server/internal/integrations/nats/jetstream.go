package nats

import (
	"context"
	"fmt"
	"time"

	natsjs "github.com/nats-io/nats.go/jetstream"
)

type StreamLimits struct {
	JobsMaxBytes   int64
	EventsMaxBytes int64
	DLQMaxBytes    int64
	JobsMaxAge     time.Duration
	EventsMaxAge   time.Duration
	DLQMaxAge      time.Duration
	Replicas       int
}

func DefaultStreamLimits() StreamLimits {
	return StreamLimits{
		JobsMaxBytes:   5 << 30,
		EventsMaxBytes: 10 << 30,
		DLQMaxBytes:    5 << 30,
		JobsMaxAge:     7 * 24 * time.Hour,
		EventsMaxAge:   30 * 24 * time.Hour,
		DLQMaxAge:      90 * 24 * time.Hour,
		Replicas:       3,
	}
}

func (c *Client) Provision(ctx context.Context, limits StreamLimits) error {
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("NATS context is required")
	}

	for _, config := range StreamConfigs(limits) {
		if _, err := c.jetStream.CreateOrUpdateStream(ctx, config); err != nil {
			return fmt.Errorf("provision NATS stream %q: %w", config.Name, err)
		}
	}

	return nil
}

func StreamConfigs(limits StreamLimits) []natsjs.StreamConfig {
	limits = normalizeStreamLimits(limits)

	return []natsjs.StreamConfig{
		newStreamConfig(
			JobsStreamName,
			"Leamout background jobs",
			JobsSubject,
			natsjs.WorkQueuePolicy,
			natsjs.DiscardNew,
			limits.JobsMaxBytes,
			limits.JobsMaxAge,
			limits.Replicas,
		),
		newStreamConfig(
			EventsStreamName,
			"Leamout domain and provider events",
			EventsSubject,
			natsjs.LimitsPolicy,
			natsjs.DiscardOld,
			limits.EventsMaxBytes,
			limits.EventsMaxAge,
			limits.Replicas,
		),
		newStreamConfig(
			DLQStreamName,
			"Leamout dead-letter messages",
			DLQSubject,
			natsjs.LimitsPolicy,
			natsjs.DiscardOld,
			limits.DLQMaxBytes,
			limits.DLQMaxAge,
			limits.Replicas,
		),
	}
}

func newStreamConfig(
	name string,
	description string,
	subject string,
	retention natsjs.RetentionPolicy,
	discard natsjs.DiscardPolicy,
	maxBytes int64,
	maxAge time.Duration,
	replicas int,
) natsjs.StreamConfig {
	return natsjs.StreamConfig{
		Name:        name,
		Description: description,
		Subjects:    []string{subject},
		Retention:   retention,
		Discard:     discard,
		Storage:     natsjs.FileStorage,
		Replicas:    replicas,
		MaxBytes:    maxBytes,
		MaxAge:      maxAge,
		MaxMsgSize:  MaxMessageSize,
		Duplicates:  10 * time.Minute,
	}
}

func normalizeStreamLimits(limits StreamLimits) StreamLimits {
	defaults := DefaultStreamLimits()

	if limits.JobsMaxBytes <= 0 {
		limits.JobsMaxBytes = defaults.JobsMaxBytes
	}
	if limits.EventsMaxBytes <= 0 {
		limits.EventsMaxBytes = defaults.EventsMaxBytes
	}
	if limits.DLQMaxBytes <= 0 {
		limits.DLQMaxBytes = defaults.DLQMaxBytes
	}
	if limits.JobsMaxAge <= 0 {
		limits.JobsMaxAge = defaults.JobsMaxAge
	}
	if limits.EventsMaxAge <= 0 {
		limits.EventsMaxAge = defaults.EventsMaxAge
	}
	if limits.DLQMaxAge <= 0 {
		limits.DLQMaxAge = defaults.DLQMaxAge
	}
	if limits.Replicas <= 0 {
		limits.Replicas = defaults.Replicas
	}

	return limits
}
