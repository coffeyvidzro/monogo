package pricing

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
}

func NewRepository(queries *sqlc.Queries, databases ...*pgxpool.Pool) *Repository {
	var db *pgxpool.Pool
	if len(databases) > 0 {
		db = databases[0]
	}

	return &Repository{
		queries: queries,
		db:      db,
	}
}

func (r *Repository) ResolveProduct(
	ctx context.Context,
	req ResolveProductRequest,
) (ProductRate, error) {
	if r.db == nil {
		return ProductRate{}, pgx.ErrNoRows
	}
	row := r.db.QueryRow(
		ctx,
		`SELECT
            mpr.id,
            mpr.organization_id,
            mpr.product,
            mpr.selector,
            mpr.currency,
            mpr.rate_micros,
            mpr.effective_at,
            mpr.expires_at
         FROM managed_product_rates AS mpr
         JOIN organizations AS o
           ON o.id = $1
          AND o.status = 'active'
          AND o.deleted_at IS NULL
         WHERE (mpr.organization_id = o.id OR mpr.organization_id IS NULL)
           AND mpr.product = $2
           AND (mpr.selector = $3 OR mpr.selector = '*')
           AND mpr.currency = $4
           AND mpr.effective_at <= $5
           AND (mpr.expires_at IS NULL OR mpr.expires_at > $5)
         ORDER BY
            (mpr.organization_id IS NOT NULL) DESC,
            (mpr.selector = $3) DESC,
            mpr.effective_at DESC
         LIMIT 1`,
		req.OrganizationID,
		req.Product,
		req.Selector,
		req.Currency,
		req.ResolvedAt,
	)
	var rate ProductRate
	err := row.Scan(
		&rate.ID,
		&rate.OrganizationID,
		&rate.Product,
		&rate.Selector,
		&rate.Currency,
		&rate.RateMicros,
		&rate.EffectiveAt,
		&rate.ExpiresAt,
	)

	return rate, err
}

func (r *Repository) Create(
	ctx context.Context,
	params sqlc.CreateCarrierRateParams,
) (sqlc.CarrierRate, error) {
	return r.queries.CreateCarrierRate(
		ctx,
		params,
	)
}

func (r *Repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.CarrierRate, error) {
	return r.queries.GetCarrierRateByID(
		ctx,
		id,
	)
}

func (r *Repository) Resolve(
	ctx context.Context,
	params sqlc.ResolveCarrierRateParams,
) (sqlc.CarrierRate, error) {
	return r.queries.ResolveCarrierRate(
		ctx,
		params,
	)
}
