package sso

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, id, organizationID uuid.UUID, req CreateRequest, secretCiphertext *string) (Connection, error) {
	row, err := r.queries.CreateSSOConnection(ctx, sqlc.CreateSSOConnectionParams{
		ID:               id,
		OrganizationID:   organizationID,
		Name:             req.Name,
		Protocol:         req.Protocol,
		Issuer:           req.Issuer,
		Configuration:    req.Configuration,
		SecretCiphertext: secretCiphertext,
	})
	if err != nil {
		return Connection{}, err
	}
	return fromConnectionRow(row), nil
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]Connection, error) {
	rows, err := r.queries.ListSSOConnections(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]Connection, 0, len(rows))
	for _, row := range rows {
		out = append(out, Connection{
			ID: row.ID, OrganizationID: row.OrganizationID, Name: row.Name,
			Protocol: row.Protocol, Issuer: row.Issuer, Configuration: row.Configuration,
			Status: row.Status, CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt),
			UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt),
		})
	}
	return out, nil
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (Connection, *string, error) {
	row, err := r.queries.GetSSOConnection(ctx, sqlc.GetSSOConnectionParams{
		ID: id, OrganizationID: organizationID,
	})
	if err != nil {
		return Connection{}, nil, err
	}
	return fromConnectionRow(row), row.SecretCiphertext, nil
}

func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest, secretCiphertext *string) (Connection, error) {
	var configuration []byte
	if req.Configuration != nil {
		configuration = *req.Configuration
	}
	row, err := r.queries.UpdateSSOConnection(ctx, sqlc.UpdateSSOConnectionParams{
		Name: req.Name, Issuer: req.Issuer, Configuration: configuration,
		SecretCiphertext: secretCiphertext, Status: req.Status,
		ID: id, OrganizationID: organizationID,
	})
	if err != nil {
		return Connection{}, err
	}
	return fromConnectionRow(row), nil
}

func (r *Repository) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DeleteSSOConnection(ctx, sqlc.DeleteSSOConnectionParams{
		ID: id, OrganizationID: organizationID,
	})
}

func fromConnectionRow(row sqlc.SsoConnection) Connection {
	return Connection{
		ID: row.ID, OrganizationID: row.OrganizationID, Name: row.Name,
		Protocol: row.Protocol, Issuer: row.Issuer, Configuration: row.Configuration,
		Status: row.Status, CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
