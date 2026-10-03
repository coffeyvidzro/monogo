package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type recordingDeleter interface {
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

type cleanupRepository interface {
	ListEnabled(context.Context) ([]Policy, error)
	ListExpiredRecordings(context.Context, uuid.UUID, time.Time, int32) ([]uuid.UUID, error)
}

type CleanupJobConfig struct {
	Interval  time.Duration
	BatchSize int32
}

func DefaultCleanupJobConfig() CleanupJobConfig {
	return CleanupJobConfig{
		Interval:  time.Hour,
		BatchSize: 100,
	}
}

type CleanupJob struct {
	repo       cleanupRepository
	recordings recordingDeleter
	config     CleanupJobConfig
	now        func() time.Time
}

func NewCleanupJob(repo cleanupRepository, recordings recordingDeleter, config CleanupJobConfig) (*CleanupJob, error) {
	if repo == nil || recordings == nil {
		return nil, fmt.Errorf("retention repository and recording service are required")
	}
	if config.Interval <= 0 || config.BatchSize <= 0 {
		return nil, fmt.Errorf("retention cleanup interval and batch size must be positive")
	}
	return &CleanupJob{
		repo:       repo,
		recordings: recordings,
		config:     config,
		now:        time.Now,
	}, nil
}
func (j *CleanupJob) Run(ctx context.Context) error {
	if err := j.runOnce(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := j.runOnce(ctx); err != nil {
				return err
			}
		}
	}
}
func (j *CleanupJob) runOnce(ctx context.Context) error {
	policies, err := j.repo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("list enabled retention policies: %w", err)
	}
	for _, policy := range policies {
		if policy.Resource != ResourceRecordings {
			continue
		}
		cutoff := j.now().UTC().AddDate(0, 0, -int(policy.RetentionDays))
		for {
			ids, err := j.repo.ListExpiredRecordings(ctx, policy.OrganizationID, cutoff, j.config.BatchSize)
			if err != nil {
				return fmt.Errorf("list expired recordings for organization %s: %w", policy.OrganizationID, err)
			}
			for _, id := range ids {
				// Recording deletion removes the object from its pinned managed/BYOS
				// destination before marking metadata deleted.
				if err := j.recordings.Delete(ctx, policy.OrganizationID, id); err != nil {
					return fmt.Errorf("delete retained recording %s: %w", id, err)
				}
			}
			if len(ids) < int(j.config.BatchSize) {
				break
			}
		}
	}
	return nil
}
