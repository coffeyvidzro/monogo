package numbers

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
	if service == nil || service.repo == nil || service.repo.queries == nil ||
		service.db == nil || service.inventory == nil {
		return nil, fmt.Errorf("managed number reconciliation dependencies are required")
	}
	if batch < 1 || batch > 500 {
		return nil, fmt.Errorf("managed number reconciliation batch must be between 1 and 500")
	}
	return &ReconciliationJob{service: service, batch: batch, interval: 30 * time.Second}, nil
}

// RunOnce is safe to call from multiple workers: lifecycle updates are
// conditional, provider ordering is never performed here, and activation is
// serialized by the durable order row.
func (j *ReconciliationJob) RunOnce(ctx context.Context) error {
	orders, err := j.service.repo.ListManagedOrdersDue(ctx, int32(j.batch))
	if err != nil {
		return fmt.Errorf("list managed number reconciliations: %w", err)
	}
	for _, order := range orders {
		if _, err := j.service.Reconcile(ctx, order.ID); err != nil {
			return fmt.Errorf("reconcile managed number order %s: %w", order.ID, err)
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
