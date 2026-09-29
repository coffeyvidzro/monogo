package checkout

import (
	"context"
	"fmt"
	"time"
)

const (
	expirationBatchSize = 500
	expirationInterval  = time.Minute
)

type ExpirationJob struct {
	service *Service
}

func NewExpirationJob(service *Service) (*ExpirationJob, error) {
	if service == nil || service.repo == nil || service.repo.queries == nil {
		return nil, fmt.Errorf("checkout expiration dependencies are required")
	}

	return &ExpirationJob{
		service: service,
	}, nil
}

func (j *ExpirationJob) RunOnce(ctx context.Context) error {
	for {
		expired, err := j.service.ExpireDue(ctx, expirationBatchSize)
		if err != nil {
			return fmt.Errorf("expire due checkouts: %w", err)
		}
		if expired < expirationBatchSize {
			return nil
		}
	}
}

func (j *ExpirationJob) Run(ctx context.Context) error {
	if err := j.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(expirationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := j.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}
