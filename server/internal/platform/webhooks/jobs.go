package webhooks

import (
	"context"
	"fmt"
	"time"
)

type DeliveryJobConfig struct {
	WorkerID         string
	BatchSize        int32
	PollInterval     time.Duration
	LockTimeout      time.Duration
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration
	MaxAttempts      int32
	AutoDisableAfter int32
}

func DefaultDeliveryJobConfig(workerID string) DeliveryJobConfig {
	return DeliveryJobConfig{
		WorkerID:         workerID,
		BatchSize:        50,
		PollInterval:     500 * time.Millisecond,
		LockTimeout:      30 * time.Second,
		RetryBaseDelay:   2 * time.Second,
		RetryMaxDelay:    5 * time.Minute,
		MaxAttempts:      8,
		AutoDisableAfter: 5,
	}
}

type DeliveryJob struct {
	repo   *Repository
	sender DeliverySender
	config DeliveryJobConfig
}

func NewDeliveryJob(repo *Repository, sender DeliverySender, config DeliveryJobConfig) (*DeliveryJob, error) {
	if repo == nil {
		return nil, fmt.Errorf("webhook repository is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("webhook delivery sender is required")
	}
	if config.WorkerID == "" {
		return nil, fmt.Errorf("webhook delivery worker id is required")
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 500 * time.Millisecond
	}
	if config.LockTimeout <= 0 {
		config.LockTimeout = 30 * time.Second
	}
	if config.RetryBaseDelay <= 0 {
		config.RetryBaseDelay = 2 * time.Second
	}
	if config.RetryMaxDelay < config.RetryBaseDelay {
		config.RetryMaxDelay = 5 * time.Minute
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 8
	}
	if config.AutoDisableAfter <= 0 {
		config.AutoDisableAfter = 5
	}
	return &DeliveryJob{repo: repo, sender: sender, config: config}, nil
}

func (j *DeliveryJob) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("webhook delivery context is required")
	}
	ticker := time.NewTicker(j.config.PollInterval)
	defer ticker.Stop()

	for {
		if err := j.deliverBatch(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (j *DeliveryJob) deliverBatch(ctx context.Context) error {
	deliveries, err := j.repo.ClaimDeliveries(ctx, j.config.WorkerID, time.Now().UTC().Add(-j.config.LockTimeout), j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("claim webhook deliveries: %w", err)
	}

	for _, delivery := range deliveries {
		if ctx.Err() != nil {
			return nil
		}
		attempt := j.sender.Send(ctx, delivery)
		if deliverySucceeded(attempt) {
			if err := j.repo.MarkDeliverySucceeded(ctx, j.config.WorkerID, delivery.ID, attempt); err != nil {
				return fmt.Errorf("mark webhook delivery %s succeeded: %w", delivery.ID, err)
			}
			continue
		}

		if retryableDelivery(attempt) && delivery.AttemptCount < j.config.MaxAttempts {
			nextAttempt := time.Now().UTC().Add(j.retryDelay(delivery.AttemptCount))
			if err := j.repo.ScheduleDeliveryRetry(ctx, j.config.WorkerID, delivery.ID, nextAttempt, attempt); err != nil {
				return fmt.Errorf("schedule webhook delivery %s retry: %w", delivery.ID, err)
			}
			continue
		}

		if err := j.repo.MarkDeliveryFailed(ctx, j.config.WorkerID, delivery.ID, attempt, j.config.AutoDisableAfter); err != nil {
			return fmt.Errorf("mark webhook delivery %s failed: %w", delivery.ID, err)
		}
	}
	return nil
}

func deliverySucceeded(attempt DeliveryAttempt) bool {
	return attempt.Err == nil && attempt.StatusCode != nil && *attempt.StatusCode >= 200 && *attempt.StatusCode < 300
}

func retryableDelivery(attempt DeliveryAttempt) bool {
	if attempt.Err != nil {
		return true
	}
	if attempt.StatusCode == nil {
		return true
	}
	status := *attempt.StatusCode
	return status == 408 || status == 425 || status == 429 || status >= 500
}

func (j *DeliveryJob) retryDelay(attempt int32) time.Duration {
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
