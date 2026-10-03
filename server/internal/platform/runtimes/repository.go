package runtimes

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

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (Runtime, error) {
	row, err := r.queries.CreateRuntime(ctx, sqlc.CreateRuntimeParams{
		OrganizationID: organizationID,
		Name:           req.Name,
		Region:         req.Region,
	})
	if err != nil {
		return Runtime{}, err
	}
	return runtimeFromRow(row), nil
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Runtime, error) {
	row, err := r.queries.GetRuntime(ctx, sqlc.GetRuntimeParams{
		OrganizationID: organizationID,
		ID:             id,
	})
	if err != nil {
		return Runtime{}, err
	}
	return runtimeFromRow(row), nil
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Runtime, error) {
	rows, err := r.queries.ListRuntimes(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]Runtime, 0, len(rows))
	for _, row := range rows {
		result = append(result, runtimeFromRow(row))
	}
	return result, nil
}

func (r *Repository) Heartbeat(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req HeartbeatRequest,
) (Runtime, error) {
	row, err := r.queries.HeartbeatRuntime(ctx, sqlc.HeartbeatRuntimeParams{
		Version:        &req.Version,
		Capabilities:   req.Capabilities,
		Capacity:       req.Capacity,
		ActiveSessions: req.ActiveSessions,
		OrganizationID: organizationID,
		ID:             id,
	})
	if err != nil {
		return Runtime{}, err
	}
	return runtimeFromRow(row), nil
}

func runtimeFromRow(row sqlc.Runtime) Runtime {
	return Runtime{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Status:         row.Status,
		Version:        row.Version,
		Region:         row.Region,
		Capabilities:   row.Capabilities,
		Capacity:       row.Capacity,
		ActiveSessions: row.ActiveSessions,
		LastSeenAt:     pgconv.TimestamptzToTimePtr(row.LastSeenAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
