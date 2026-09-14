package numbers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type ProviderOperationJobConfig struct {
	Interval  time.Duration
	BatchSize int32
}

func DefaultProviderOperationJobConfig() ProviderOperationJobConfig {
	return ProviderOperationJobConfig{Interval: 10 * time.Second, BatchSize: 50}
}

type providerOperationJobRepository interface {
	ListProviderOperationsReady(context.Context, int32) ([]sqlc.ProviderOperation, error)
	TryProviderOperationLock(context.Context, uuid.UUID) (func(), bool, error)
}

type ProviderOperationExecutor interface {
	ExecuteProviderOperation(context.Context, sqlc.ProviderOperation) error
}

type ProviderOperationJob struct {
	repo     providerOperationJobRepository
	executor ProviderOperationExecutor
	config   ProviderOperationJobConfig
}

func NewProviderOperationJob(repo providerOperationJobRepository, executor ProviderOperationExecutor, config ProviderOperationJobConfig) (*ProviderOperationJob, error) {
	if repo == nil {
		return nil, fmt.Errorf("provider operation repository is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("provider operation executor is required")
	}
	if config.Interval <= 0 {
		config.Interval = 10 * time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	return &ProviderOperationJob{repo: repo, executor: executor, config: config}, nil
}

func (j *ProviderOperationJob) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("provider operation context is required")
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

func (j *ProviderOperationJob) runPass(ctx context.Context) {
	if err := j.Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("provider operation pass failed: %v", err)
	}
}

func (j *ProviderOperationJob) Execute(ctx context.Context) error {
	operations, err := j.repo.ListProviderOperationsReady(ctx, j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("list provider operations ready for execution: %w", err)
	}
	var firstErr error
	for _, operation := range operations {
		if operation.OperationType != "number_provision" {
			continue
		}
		release, locked, err := j.repo.TryProviderOperationLock(ctx, operation.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("lock provider operation %s: %w", operation.ID, err)
			}
			continue
		}
		if !locked {
			continue
		}
		err = j.executor.ExecuteProviderOperation(ctx, operation)
		release()
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("execute provider operation %s: %w", operation.ID, err)
		}
	}
	return firstErr
}
