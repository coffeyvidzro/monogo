package storage

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
	id uuid.UUID,
	organizationID uuid.UUID,
	req CreateRequest,
	ciphertext string,
) (Integration, error) {
	row, err := r.queries.CreateStorageIntegration(
		ctx,
		sqlc.CreateStorageIntegrationParams{
			ID:                        id,
			OrganizationID:            organizationID,
			Name:                      req.Name,
			EndpointUrl:               req.EndpointURL,
			Region:                    req.Region,
			Bucket:                    req.Bucket,
			AccessKeyID:               req.AccessKeyID,
			SecretAccessKeyCiphertext: ciphertext,
			UsePathStyle:              req.UsePathStyle,
		},
	)
	if err != nil {
		return Integration{}, err
	}
	return integrationFromRow(row), nil
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Integration, error) {
	rows, err := r.queries.ListStorageIntegrations(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]Integration, 0, len(rows))
	for _, row := range rows {
		result = append(result, integrationFromRow(row))
	}
	return result, nil
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Integration, string, error) {
	row, err := r.queries.GetStorageIntegration(
		ctx,
		sqlc.GetStorageIntegrationParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		return Integration{}, "", err
	}
	return integrationFromRow(row), row.SecretAccessKeyCiphertext, nil
}

func (r *Repository) GetActiveRecording(
	ctx context.Context,
	organizationID uuid.UUID,
) (Integration, string, error) {
	row, err := r.queries.GetActiveRecordingStorageIntegration(ctx, organizationID)
	if err != nil {
		return Integration{}, "", err
	}
	return integrationFromRow(row), row.SecretAccessKeyCiphertext, nil
}

func (r *Repository) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req UpdateRequest,
	ciphertext *string,
) (Integration, error) {
	row, err := r.queries.UpdateStorageIntegration(
		ctx,
		sqlc.UpdateStorageIntegrationParams{
			Name:                      req.Name,
			AccessKeyID:               req.AccessKeyID,
			SecretAccessKeyCiphertext: ciphertext,
			Status:                    req.Status,
			ID:                        id,
			OrganizationID:            organizationID,
		},
	)
	if err != nil {
		return Integration{}, err
	}
	return integrationFromRow(row), nil
}

func (r *Repository) Disable(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Integration, error) {
	row, err := r.queries.DisableStorageIntegration(
		ctx,
		sqlc.DisableStorageIntegrationParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		return Integration{}, err
	}
	return integrationFromRow(row), nil
}

func integrationFromRow(row sqlc.StorageIntegration) Integration {
	return Integration{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Provider:       row.Provider,
		Purpose:        row.Purpose,
		EndpointURL:    row.EndpointUrl,
		Region:         row.Region,
		Bucket:         row.Bucket,
		AccessKeyID:    row.AccessKeyID,
		UsePathStyle:   row.UsePathStyle,
		Status:         row.Status,
		HasSecret:      row.SecretAccessKeyCiphertext != "",
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
