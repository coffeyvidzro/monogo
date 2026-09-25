package lifecycle

import (
	"context"
	"fmt"
	"time"
)

type ReconciliationJob struct {
	service  *Service
	batch    int
	interval time.Duration
}

func NewReconciliationJob(service *Service, batch int) (*ReconciliationJob, error) {
	if service == nil || service.repo == nil || service.repo.queries == nil || service.db == nil || service.lifecycle == nil {
		return nil, fmt.Errorf("number lifecycle reconciliation dependencies are required")
	}
	if batch < 1 || batch > 500 {
		return nil, fmt.Errorf("number lifecycle reconciliation batch must be between 1 and 500")
	}
	return &ReconciliationJob{service: service, batch: batch, interval: 30 * time.Second}, nil
}

func (j *ReconciliationJob) RunOnce(ctx context.Context) error {
	operations, err := j.service.repo.ListLifecycleDue(ctx, int32(j.batch))
	if err != nil {
		return fmt.Errorf("list number lifecycle reconciliations: %w", err)
	}
	for _, operation := range operations {
		if err := j.service.ReconcileLifecycle(ctx, operation); err != nil {
			return fmt.Errorf("reconcile number lifecycle operation %s: %w", operation.ID, err)
		}
	}
	registrations, err := j.service.repo.ListEmergencyDue(ctx, int32(j.batch))
	if err != nil {
		return fmt.Errorf("list emergency registration reconciliations: %w", err)
	}
	for _, registration := range registrations {
		if err := j.service.ReconcileEmergency(ctx, registration); err != nil {
			return fmt.Errorf("reconcile emergency registration %s: %w", registration.ID, err)
		}
	}
	return nil
}

func (j *ReconciliationJob) Run(ctx context.Context) error {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		if err := j.RunOnce(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
