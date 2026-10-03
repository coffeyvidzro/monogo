package retention

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}
func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]Policy, error) {
	rows, err := r.queries.ListRetentionPolicies(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]Policy, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}
func (r *Repository) Upsert(ctx context.Context, organizationID uuid.UUID, resource string, req UpsertRequest) (Policy, error) {
	row, err := r.queries.UpsertRetentionPolicy(ctx, sqlc.UpsertRetentionPolicyParams{
		OrganizationID: organizationID,
		Resource:       resource,
		RetentionDays:  req.RetentionDays,
		Enabled:        req.Enabled,
	})
	if err != nil {
		return Policy{}, err
	}
	return fromRow(row), nil
}
func (r *Repository) Delete(ctx context.Context, organizationID uuid.UUID, resource string) error {
	return r.queries.DeleteRetentionPolicy(ctx, sqlc.DeleteRetentionPolicyParams{
		OrganizationID: organizationID,
		Resource:       resource,
	})
}
func (r *Repository) ListEnabled(ctx context.Context) ([]Policy, error) {
	rows, err := r.queries.ListEnabledRetentionPolicies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Policy, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}
func (r *Repository) ListExpiredRecordings(ctx context.Context, organizationID uuid.UUID, before time.Time, batchSize int32) ([]uuid.UUID, error) {
	return r.queries.ListExpiredRecordingsForRetention(ctx, sqlc.ListExpiredRecordingsForRetentionParams{
		OrganizationID: organizationID,
		CreatedBefore:  timestamp(before),
		BatchSize:      batchSize,
	})
}
func (r *Repository) ListExpiredConversations(ctx context.Context, organizationID uuid.UUID, before time.Time, batchSize int32) ([]uuid.UUID, error) {
	return r.queries.ListExpiredConversationsForRetention(ctx, sqlc.ListExpiredConversationsForRetentionParams{
		OrganizationID: organizationID,
		CreatedBefore:  timestamp(before),
		BatchSize:      batchSize,
	})
}
func (r *Repository) DeleteConversation(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DeleteConversationForRetention(ctx, sqlc.DeleteConversationForRetentionParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}
func (r *Repository) ListExpiredAuditEvents(ctx context.Context, organizationID uuid.UUID, before time.Time, batchSize int32) ([]uuid.UUID, error) {
	return r.queries.ListExpiredAuditEventsForRetention(ctx, sqlc.ListExpiredAuditEventsForRetentionParams{
		OrganizationID: organizationID,
		CreatedBefore:  timestamp(before),
		BatchSize:      batchSize,
	})
}
func (r *Repository) DeleteAuditEvent(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DeleteAuditEventForRetention(ctx, sqlc.DeleteAuditEventForRetentionParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}
func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
func fromRow(row sqlc.RetentionPolicy) Policy {
	return Policy{
		OrganizationID: row.OrganizationID,
		Resource:       row.Resource,
		RetentionDays:  row.RetentionDays,
		Enabled:        row.Enabled,
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
