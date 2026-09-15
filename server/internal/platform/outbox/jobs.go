package outbox

import (
	"context"
	"fmt"
	"time"
)

type PublisherJobConfig struct {
	WorkerID          string
	BatchSize         int32
	PollInterval      time.Duration
	LockTimeout       time.Duration
	StaleLockInterval time.Duration
	RetryBaseDelay    time.Duration
	RetryMaxDelay     time.Duration
}

func DefaultPublisherJobConfig(workerID string) PublisherJobConfig {
	return PublisherJobConfig{
		WorkerID:          workerID,
		BatchSize:         100,
		PollInterval:      500 * time.Millisecond,
		LockTimeout:       60 * time.Second,
		StaleLockInterval: 30 * time.Second,
		RetryBaseDelay:    2 * time.Second,
		RetryMaxDelay:     5 * time.Minute,
	}
}

type PublisherJob struct {
	repo      *Repository
	publisher *Publisher
	config    PublisherJobConfig
}

func NewPublisherJob(repo *Repository, publisher *Publisher, config PublisherJobConfig) (*PublisherJob, error) {
	if repo == nil {
		return nil, fmt.Errorf("outbox repository is required")
	}
	if publisher == nil {
		return nil, fmt.Errorf("outbox publisher is required")
	}
	if config.WorkerID == "" {
		return nil, fmt.Errorf("outbox publisher worker id is required")
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 500 * time.Millisecond
	}
	if config.LockTimeout <= 0 {
		config.LockTimeout = 60 * time.Second
	}
	if config.StaleLockInterval <= 0 {
		config.StaleLockInterval = 30 * time.Second
	}
	if config.RetryBaseDelay <= 0 {
		config.RetryBaseDelay = 2 * time.Second
	}
	if config.RetryMaxDelay < config.RetryBaseDelay {
		config.RetryMaxDelay = 5 * time.Minute
	}
	return &PublisherJob{repo: repo, publisher: publisher, config: config}, nil
}

func (j *PublisherJob) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("outbox publisher context is required")
	}
	if err := j.releaseStaleLocks(ctx); err != nil {
		return err
	}

	pollTicker := time.NewTicker(j.config.PollInterval)
	defer pollTicker.Stop()
	staleTicker := time.NewTicker(j.config.StaleLockInterval)
	defer staleTicker.Stop()

	for {
		if err := j.publishBatch(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-pollTicker.C:
		case <-staleTicker.C:
			if err := j.releaseStaleLocks(ctx); err != nil {
				return err
			}
		}
	}
}

func (j *PublisherJob) publishBatch(ctx context.Context) error {
	events, err := j.repo.ClaimPending(ctx, j.config.WorkerID, j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("claim pending outbox events: %w", err)
	}

	for _, event := range events {
		if err := j.publisher.Publish(ctx, event); err != nil {
			delay := j.retryDelay(event.Attempts)
			if markErr := j.repo.MarkFailed(ctx, event.ID, err, durationSeconds(delay)); markErr != nil {
				return fmt.Errorf("mark outbox event %s failed after publish error: %w", event.ID, markErr)
			}
			continue
		}
		if err := j.repo.MarkPublished(ctx, event.ID); err != nil {
			return fmt.Errorf("mark outbox event %s published: %w", event.ID, err)
		}
	}
	return nil
}

func (j *PublisherJob) releaseStaleLocks(ctx context.Context) error {
	if err := j.repo.ReleaseStaleLocks(ctx, durationSeconds(j.config.LockTimeout)); err != nil {
		return fmt.Errorf("release stale outbox locks: %w", err)
	}
	return nil
}

func (j *PublisherJob) retryDelay(attempt int32) time.Duration {
	delay := j.config.RetryBaseDelay
	for i := int32(1); i < attempt && delay < j.config.RetryMaxDelay; i++ {
		delay *= 2
		if delay >= j.config.RetryMaxDelay {
			return j.config.RetryMaxDelay
		}
	}
	if delay > j.config.RetryMaxDelay {
		return j.config.RetryMaxDelay
	}
	return delay
}

func durationSeconds(v time.Duration) int32 {
	seconds := int64(v / time.Second)
	if seconds < 1 {
		return 1
	}
	if seconds > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(seconds)
}
