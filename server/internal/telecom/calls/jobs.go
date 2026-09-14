package calls

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

const reconciliationHangupReason = "RECONCILIATION_CHANNEL_MISSING"

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
	ListForReconciliation(context.Context, time.Time, int32) ([]sqlc.Call, error)
	MarkCompleted(context.Context, uuid.UUID, uuid.UUID, *string) (sqlc.Call, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, *string) (sqlc.Call, error)
}

type channelInventory interface {
	Channels(context.Context) ([]freeswitch.Channel, error)
}

type ReconciliationJob struct {
	repo      reconciliationRepository
	inventory channelInventory
	config    ReconciliationJobConfig
	now       func() time.Time
	admission *AdmissionController
	metrics   TelecomMetrics
}

func (j *ReconciliationJob) SetMetrics(metrics TelecomMetrics) { j.metrics = metrics }

func NewReconciliationJob(
	repo reconciliationRepository,
	inventory channelInventory,
	config ReconciliationJobConfig,
	admission ...*AdmissionController,
) (*ReconciliationJob, error) {
	if repo == nil {
		return nil, fmt.Errorf("call reconciliation repository is required")
	}
	if inventory == nil {
		return nil, fmt.Errorf("call reconciliation channel inventory is required")
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

	job := &ReconciliationJob{
		repo:      repo,
		inventory: inventory,
		config:    config,
		now:       time.Now,
	}
	if len(admission) > 0 {
		job.admission = admission[0]
	}
	return job, nil
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
	channels, err := j.inventory.Channels(ctx)
	if err != nil {
		return fmt.Errorf("list FreeSWITCH channels: %w", err)
	}

	active := make(map[string]struct{}, len(channels))
	for _, channel := range channels {
		if channel.UUID != "" {
			active[channel.UUID] = struct{}{}
		}
	}

	updatedBefore := j.now().UTC().Add(-j.config.Grace)
	stale, err := j.repo.ListForReconciliation(ctx, updatedBefore, j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("list calls for reconciliation: %w", err)
	}

	reason := reconciliationHangupReason
	for _, call := range stale {
		if call.SipCallID == nil || *call.SipCallID == "" {
			continue
		}
		if _, ok := active[*call.SipCallID]; ok {
			if j.admission != nil && call.CarrierConnectionID != nil {
				if err := j.admission.Refresh(ctx, *call.CarrierConnectionID, *call.SipCallID); err != nil {
					return fmt.Errorf("refresh active call %s quota lease: %w", call.ID, err)
				}
			}
			continue
		}

		if j.admission != nil && call.CarrierConnectionID != nil {
			if err := j.admission.Release(ctx, *call.CarrierConnectionID, *call.SipCallID); err != nil {
				return fmt.Errorf("release reconciled call %s quota lease: %w", call.ID, err)
			}
		}

		switch CallState(call.State) {
		case StateAnswered, StateActive:
			_, err = j.repo.MarkCompleted(ctx, call.OrganizationID, call.ID, &reason)
		case StateInitiating, StateRinging:
			_, err = j.repo.MarkFailed(ctx, call.OrganizationID, call.ID, &reason)
		default:
			continue
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("reconcile call %s: %w", call.ID, err)
		}
		if err == nil && j.metrics != nil && call.CarrierConnectionID != nil {
			state := "failed"
			if CallState(call.State) == StateAnswered || CallState(call.State) == StateActive {
				state = "completed"
			}
			j.metrics.Call(ctx, state, *call.CarrierConnectionID, uuidValue(call.TrunkID), uuidValue(call.TrunkEndpointID))
		}
	}

	return nil
}
