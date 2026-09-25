package lifecycle

import (
	"context"
	"errors"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

const managedReconcileDelay = 30 * time.Second

type Service struct {
	repo      *Repository
	db        *pgxpool.Pool
	lifecycle NumberLifecycleProvider
	now       func() time.Time
}

func NewService(repo *Repository, db *pgxpool.Pool, provider NumberLifecycleProvider) *Service {
	return &Service{repo: repo, db: db, lifecycle: provider, now: time.Now}
}

func (s *Service) GetLifecycle(ctx context.Context, organizationID, operationID uuid.UUID) (sqlc.NumberLifecycleOperation, error) {
	op, err := s.repo.GetLifecycleOperation(ctx, organizationID, operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.NumberLifecycleOperation{}, apperror.NewNotFound("number lifecycle operation not found")
	}
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("get number lifecycle operation", err)
	}
	return op, nil
}

func (s *Service) ReconcileLifecycle(ctx context.Context, operation sqlc.NumberLifecycleOperation) error {
	switch operation.Operation {
	case "release":
		return s.reconcileRelease(ctx, operation)
	case "port_in":
		return s.reconcilePortIn(ctx, operation)
	default:
		return nil
	}
}
