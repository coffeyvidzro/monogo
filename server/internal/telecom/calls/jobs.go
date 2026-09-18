package calls

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type ReconciliationJobConfig struct {
	Interval time.Duration
}

func DefaultReconciliationJobConfig() ReconciliationJobConfig {
	return ReconciliationJobConfig{
		Interval: 30 * time.Second,
	}
}

type reconciliationRepository interface {
	ListActiveForAdmissionReconciliation(
		context.Context,
	) ([]sqlc.ListActiveCallsForAdmissionReconciliationRow, error)
}

type reconciliationAdmission interface {
	Refresh(context.Context, uuid.UUID, uuid.UUID) error
}

type ReconciliationJob struct {
	repo      reconciliationRepository
	admission reconciliationAdmission
	config    ReconciliationJobConfig
}

func NewReconciliationJob(
	repo reconciliationRepository,
	admission reconciliationAdmission,
	config ReconciliationJobConfig,
) (*ReconciliationJob, error) {
	if repo == nil {
		return nil, fmt.Errorf("call reconciliation repository is required")
	}
	if admission == nil {
		return nil, fmt.Errorf("call reconciliation admission limiter is required")
	}
	if config.Interval <= 0 {
		config.Interval = 30 * time.Second
	}

	return &ReconciliationJob{
		repo:      repo,
		admission: admission,
		config:    config,
	}, nil
}

func (j *ReconciliationJob) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("call reconciliation context is required")
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
		log.Printf("call reconciliation pass failed: %v", err)
	}
}

func (j *ReconciliationJob) Reconcile(ctx context.Context) error {
	active, err := j.repo.ListActiveForAdmissionReconciliation(ctx)
	if err != nil {
		return fmt.Errorf("list calls for admission reconciliation: %w", err)
	}

	for _, call := range active {
		if call.CarrierConnectionID == nil {
			continue
		}
		if err := j.admission.Refresh(ctx, *call.CarrierConnectionID, call.ID); err != nil {
			return fmt.Errorf("refresh admission lease for call %s: %w", call.ID, err)
		}
	}

	return nil
}
