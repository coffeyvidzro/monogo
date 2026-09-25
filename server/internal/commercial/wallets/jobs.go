package wallets

import (
	"context"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type expirationRepository interface {
	ExpiredReservations(context.Context, int32) ([]sqlc.WalletReservation, error)
}

type expirationService interface {
	Expire(context.Context, uuid.UUID, uuid.UUID) (sqlc.WalletReservation, error)
}

type ExpirationJob struct {
	repository expirationRepository
	service    expirationService
	batch      int32
	interval   time.Duration
}

func NewExpirationJob(repository expirationRepository, service expirationService, batch int32, interval time.Duration) (*ExpirationJob, error) {
	if repository == nil || service == nil {
		return nil, fmt.Errorf("reservation expiration dependencies are required")
	}
	if batch < 1 || batch > 500 {
		return nil, fmt.Errorf("reservation expiration batch must be between 1 and 500")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("reservation expiration interval must be positive")
	}
	return &ExpirationJob{repository: repository, service: service, batch: batch, interval: interval}, nil
}

func (j *ExpirationJob) RunOnce(ctx context.Context) error {
	rows, err := j.repository.ExpiredReservations(ctx, j.batch)
	if err != nil {
		return fmt.Errorf("list expired wallet reservations: %w", err)
	}
	for _, row := range rows {
		if _, err := j.service.Expire(ctx, row.OrganizationID, row.ID); err != nil {
			return fmt.Errorf("expire wallet reservation %s: %w", row.ID, err)
		}
	}
	return nil
}

func (j *ExpirationJob) Run(ctx context.Context) error {
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
