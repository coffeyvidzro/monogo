package checkout

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repo: NewRepository(db)}
}

// Create only records a top-up intention: it cannot charge a provider or
// modify any wallet balance.
func (s *Service) Create(ctx context.Context, req CreateRequest) (sqlc.Checkout, error) {
	if err := validateCreate(&req); err != nil {
		return sqlc.Checkout{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Checkout{}, apperror.NewServiceUnavailable("checkout persistence is not configured", nil)
	}

	hash := requestHash(req)
	row, err := s.repo.Create(ctx, req, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = s.repo.ByKey(ctx, req)
		if err == nil && row.RequestHash != hash {
			return sqlc.Checkout{}, apperror.NewConflict("checkout idempotency key was used for a different request")
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Checkout{}, apperror.NewConflict("wallet unavailable or checkout expired")
	}
	if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("create checkout", err)
	}
	return row, nil
}

func (s *Service) Get(ctx context.Context, organizationID, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	if organizationID == uuid.Nil || checkoutID == uuid.Nil {
		return sqlc.Checkout{}, apperror.NewBadRequest("organization and checkout are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Checkout{}, apperror.NewServiceUnavailable("checkout persistence is not configured", nil)
	}
	row, err := s.repo.Get(ctx, organizationID, checkoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Checkout{}, apperror.NewNotFound("checkout not found")
	}
	if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("get checkout", err)
	}
	return row, nil
}

func (s *Service) Cancel(ctx context.Context, organizationID, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	if organizationID == uuid.Nil || checkoutID == uuid.Nil {
		return sqlc.Checkout{}, apperror.NewBadRequest("organization and checkout are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Checkout{}, apperror.NewServiceUnavailable("checkout persistence is not configured", nil)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("begin checkout cancellation", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repo := s.repo.WithTx(tx)
	if _, err = repo.Get(ctx, organizationID, checkoutID); errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Checkout{}, apperror.NewNotFound("checkout not found")
	} else if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("get checkout", err)
	}
	current, err := repo.Lock(ctx, checkoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Checkout{}, apperror.NewNotFound("checkout not found")
	}
	if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("get checkout", err)
	}
	if current.Status == "canceled" {
		return current, nil
	}
	if current.Status != "pending" {
		return sqlc.Checkout{}, apperror.NewConflict("checkout cannot be canceled")
	}
	row, err := repo.Cancel(ctx, checkoutID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Checkout{}, apperror.NewConflict("checkout has an active payment and cannot be canceled")
	}
	if err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("cancel checkout", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Checkout{}, apperror.NewInternal("commit checkout cancellation", err)
	}
	return row, nil
}
