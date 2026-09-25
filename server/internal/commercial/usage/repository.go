package usage

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries, databases ...*pgxpool.Pool) *Repository {
	var db *pgxpool.Pool
	if len(databases) > 0 {
		db = databases[0]
	}
	return &Repository{db: db, queries: queries}
}

func (r *Repository) Available() bool {
	return r != nil && r.queries != nil
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) { return r.db.Begin(ctx) }

func (r *Repository) WithTx(tx pgx.Tx) *Repository { return &Repository{queries: r.queries.WithTx(tx)} }

func (r *Repository) Record(ctx context.Context, req RecordRequest) (sqlc.UsageEvent, error) {
	return r.queries.CreateUsageEvent(ctx, sqlc.CreateUsageEventParams{
		OrganizationID: req.OrganizationID,
		MeterID:        req.MeterID,
		Quantity:       req.Quantity,
		SourceType:     req.SourceType,
		SourceID:       req.SourceID,
		IdempotencyKey: req.IdempotencyKey,
		Dimensions:     []byte(req.Dimensions),
		OccurredAt:     pgconv.TimeToTimestamptz(req.OccurredAt),
	})
}

func (r *Repository) ByKey(ctx context.Context, organizationID uuid.UUID, key string) (sqlc.UsageEvent, error) {
	return r.queries.GetUsageEventByKey(ctx, sqlc.GetUsageEventByKeyParams{
		OrganizationID: organizationID,
		IdempotencyKey: key,
	})
}

func (r *Repository) MatchingByKey(ctx context.Context, req RecordRequest) (sqlc.UsageEvent, error) {
	return r.queries.GetMatchingUsageEventByKey(ctx, sqlc.GetMatchingUsageEventByKeyParams{
		OrganizationID: req.OrganizationID,
		IdempotencyKey: req.IdempotencyKey,
		MeterID:        req.MeterID,
		Quantity:       req.Quantity,
		SourceType:     req.SourceType,
		SourceID:       req.SourceID,
		Dimensions:     []byte(req.Dimensions),
		OccurredAt:     pgconv.TimeToTimestamptz(req.OccurredAt),
	})
}

func (r *Repository) Get(ctx context.Context, organizationID, eventID uuid.UUID) (sqlc.UsageEvent, error) {
	return r.queries.GetUsageEvent(ctx, sqlc.GetUsageEventParams{
		OrganizationID: organizationID,
		ID:             eventID,
	})
}
