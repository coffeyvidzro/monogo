package settlement

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(
	db *pgxpool.Pool,
	queries *sqlc.Queries,
) *Repository {
	if db == nil {
		panic("billing settlement: database is required")
	}
	if queries == nil {
		panic("billing settlement: queries are required")
	}

	return &Repository{
		db:      db,
		queries: queries,
	}
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, *sqlc.Queries, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin OCS persistence transaction: %w", err)
	}

	return tx, r.queries.WithTx(tx), nil
}
