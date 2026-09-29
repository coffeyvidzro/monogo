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

// RunOnce reconciles managed number orders without executing lifecycle operations.
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

type RenewalJob struct {
	service  *Service
	batch    int
	interval time.Duration
}

func NewRenewalJob(service *Service, batch int) (*RenewalJob, error) {
	if service == nil || service.repo == nil || service.repo.queries == nil ||
		service.pricing == nil || service.wallets == nil {
		return nil, fmt.Errorf("managed number renewal dependencies are required")
	}
	if batch < 1 || batch > 500 {
		return nil, fmt.Errorf("managed number renewal batch must be between 1 and 500")
	}
	return &RenewalJob{
		service:  service,
		batch:    batch,
		interval: time.Hour,
	}, nil
}

func (j *RenewalJob) RunOnce(ctx context.Context) error {
	now := j.service.now().UTC()
	if err := j.service.repo.ScheduleRenewals(ctx, now); err != nil {
		return fmt.Errorf("schedule managed number renewals: %w", err)
	}
	renewals, err := j.service.repo.ListRenewalsDue(ctx, now, int32(j.batch))
	if err != nil {
		return fmt.Errorf("list managed number renewals: %w", err)
	}
	for _, renewal := range renewals {
		if err := j.service.processRenewal(ctx, renewal); err != nil {
			return fmt.Errorf("process managed number renewal %s: %w", renewal.ID, err)
		}
	}
	return nil
}

func (j *RenewalJob) Run(ctx context.Context) error {
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
