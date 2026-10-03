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
	ListExpiredConversations(context.Context, uuid.UUID, time.Time, int32) ([]uuid.UUID, error)
	DeleteConversation(context.Context, uuid.UUID, uuid.UUID) error
	ListExpiredAuditEvents(context.Context, uuid.UUID, time.Time, int32) ([]uuid.UUID, error)
	DeleteAuditEvent(context.Context, uuid.UUID, uuid.UUID) error
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
		cutoff := j.now().UTC().AddDate(0, 0, -int(policy.RetentionDays))
		switch policy.Resource {
		case ResourceRecordings:
			if err := j.cleanupRecordings(ctx, policy.OrganizationID, cutoff); err != nil {
				return err
			}
		case ResourceConversations:
			if err := j.cleanupConversations(ctx, policy.OrganizationID, cutoff); err != nil {
				return err
			}
		case ResourceAuditEvents:
			if err := j.cleanupAuditEvents(ctx, policy.OrganizationID, cutoff); err != nil {
				return err
			}
		}
	}
	return nil
}

func (j *CleanupJob) cleanupRecordings(ctx context.Context, organizationID uuid.UUID, cutoff time.Time) error {
	for {
		ids, err := j.repo.ListExpiredRecordings(ctx, organizationID, cutoff, j.config.BatchSize)
		if err != nil {
			return fmt.Errorf("list expired recordings for organization %s: %w", organizationID, err)
		}
		for _, id := range ids {
			// Recording deletion removes the object from its pinned managed/BYOS
			// destination before marking metadata deleted.
			if err := j.recordings.Delete(ctx, organizationID, id); err != nil {
				return fmt.Errorf("delete retained recording %s: %w", id, err)
			}
		}
		if len(ids) < int(j.config.BatchSize) {
			return nil
		}
	}
}

func (j *CleanupJob) cleanupConversations(ctx context.Context, organizationID uuid.UUID, cutoff time.Time) error {
	for {
		ids, err := j.repo.ListExpiredConversations(ctx, organizationID, cutoff, j.config.BatchSize)
		if err != nil {
			return fmt.Errorf("list expired conversations for organization %s: %w", organizationID, err)
		}
		for _, id := range ids {
			if err := j.repo.DeleteConversation(ctx, organizationID, id); err != nil {
				return fmt.Errorf("delete retained conversation %s: %w", id, err)
			}
		}
		if len(ids) < int(j.config.BatchSize) {
			return nil
		}
	}
}

func (j *CleanupJob) cleanupAuditEvents(ctx context.Context, organizationID uuid.UUID, cutoff time.Time) error {
	for {
		ids, err := j.repo.ListExpiredAuditEvents(ctx, organizationID, cutoff, j.config.BatchSize)
		if err != nil {
			return fmt.Errorf("list expired audit events for organization %s: %w", organizationID, err)
		}
		for _, id := range ids {
			if err := j.repo.DeleteAuditEvent(ctx, organizationID, id); err != nil {
				return fmt.Errorf("delete retained audit event %s: %w", id, err)
			}
		}
		if len(ids) < int(j.config.BatchSize) {
			return nil
		}
	}
}
