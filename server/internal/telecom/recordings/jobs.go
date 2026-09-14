package recordings

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type ReconciliationJobConfig struct {
	Interval  time.Duration
	Grace     time.Duration
	BatchSize int32
}

func DefaultReconciliationJobConfig() ReconciliationJobConfig {
	return ReconciliationJobConfig{
		Interval:  30 * time.Second,
		Grace:     60 * time.Second,
		BatchSize: 200,
	}
}

type reconciliationRepository interface {
	ListForReconciliation(context.Context, time.Time, int32) ([]sqlc.Recording, error)
	Complete(context.Context, sqlc.Recording) (sqlc.Recording, error)
}

type ReconciliationJob struct {
	repo   reconciliationRepository
	config ReconciliationJobConfig
	now    func() time.Time
}

func NewReconciliationJob(
	repo reconciliationRepository,
	config ReconciliationJobConfig,
) (*ReconciliationJob, error) {
	if repo == nil {
		return nil, fmt.Errorf("recording reconciliation repository is required")
	}
	if config.Interval <= 0 {
		config.Interval = 30 * time.Second
	}
	if config.Grace <= 0 {
		config.Grace = 60 * time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 200
	}

	return &ReconciliationJob{
		repo:   repo,
		config: config,
		now:    time.Now,
	}, nil
}

func (j *ReconciliationJob) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("recording reconciliation context is required")
	}

	j.runPass(ctx)

	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			j.runPass(ctx)
		}
	}
}

func (j *ReconciliationJob) runPass(ctx context.Context) {
	if err := j.Reconcile(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("recording reconciliation pass failed: %v", err)
	}
}

func (j *ReconciliationJob) Reconcile(ctx context.Context) error {
	updatedBefore := j.now().UTC().Add(-j.config.Grace)
	stale, err := j.repo.ListForReconciliation(ctx, updatedBefore, j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("list recordings for reconciliation: %w", err)
	}

	for _, recording := range stale {
		_, err := j.repo.Complete(ctx, recording)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("reconcile recording %s: %w", recording.ID, err)
		}
	}

	return nil
}
