package usage

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Available() bool {
	return r != nil && r.queries != nil
}

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
