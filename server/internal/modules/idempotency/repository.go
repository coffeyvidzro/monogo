package idempotency

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Claim(ctx context.Context, params sqlc.ClaimIdempotencyKeyParams) (sqlc.Idempotency, bool, error) {
	record, err := r.queries.ClaimIdempotencyKey(ctx, params)
	if err == nil {
		return record, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Idempotency{}, false, err
	}
	record, err = r.queries.GetIdempotencyKey(ctx, sqlc.GetIdempotencyKeyParams{
		Scope: params.Scope, IdempotencyKey: params.IdempotencyKey,
	})
	return record, false, err
}

func (r *Repository) Complete(ctx context.Context, params sqlc.CompleteIdempotencyKeyParams) error {
	_, err := r.queries.CompleteIdempotencyKey(ctx, params)
	return err
}

func (r *Repository) DeleteExpired(ctx context.Context) (int64, error) {
	return r.queries.DeleteExpiredIdempotencyKeys(ctx)
}
