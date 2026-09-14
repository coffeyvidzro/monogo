package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	if db == nil {
		panic("audit: database is required")
	}
	return &Repository{queries: sqlc.New(db)}
}

func Insert(ctx context.Context, tx pgx.Tx, event Event) error {
	if tx == nil {
		return fmt.Errorf("audit transaction is required")
	}
	return sqlc.New(tx).InsertAuditEvent(ctx, sqlc.InsertAuditEventParams{
		OrganizationID: event.OrganizationID,
		ActorType:      event.ActorType,
		ActorID:        event.ActorID,
		Action:         event.Action,
		TargetType:     event.TargetType,
		TargetID:       event.TargetID,
		Metadata:       event.Metadata,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID, limit, offset int32) ([]Event, error) {
	rows, err := r.queries.ListAuditEventsByOrganizationID(ctx, sqlc.ListAuditEventsByOrganizationIDParams{
		OrganizationID: organizationID,
		LimitCount:     limit,
		OffsetCount:    offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Event, 0, len(rows))
	for _, row := range rows {
		items = append(items, Event{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			ActorType:      row.ActorType,
			ActorID:        row.ActorID,
			Action:         row.Action,
			TargetType:     row.TargetType,
			TargetID:       row.TargetID,
			Metadata:       row.Metadata,
			OccurredAt:     pgconv.TimestamptzToTime(row.OccurredAt),
		})
	}
	return items, nil
}
