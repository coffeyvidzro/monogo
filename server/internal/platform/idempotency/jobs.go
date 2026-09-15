package idempotency

import (
	"context"
	"fmt"
	"time"
)

type CleanupJob struct {
	repository *Repository
	interval   time.Duration
}

func NewCleanupJob(repository *Repository, interval time.Duration) (*CleanupJob, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("idempotency cleanup interval must be positive")
	}
	return &CleanupJob{repository: repository, interval: interval}, nil
}

func (j *CleanupJob) Run(ctx context.Context) error {
	if _, err := j.repository.DeleteExpired(ctx); err != nil {
		return fmt.Errorf("delete expired idempotency keys: %w", err)
	}
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := j.repository.DeleteExpired(ctx); err != nil {
				return fmt.Errorf("delete expired idempotency keys: %w", err)
			}
		}
	}
}
