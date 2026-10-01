package numbers

import (
	"context"

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
) (sqlc.PhoneNumber, error) {
	return r.queries.CreateBYOCPhoneNumber(
		ctx,
		sqlc.CreateBYOCPhoneNumberParams{
			OrganizationID: organizationID,
			Number:         req.Number,
			CountryCode:    req.CountryCode,
			TrunkID:        req.TrunkID,
			VoiceEnabled:   req.VoiceEnabled,
		},
	)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.PhoneNumber, error) {
	return r.queries.ListPhoneNumbersByOrganizationID(
		ctx,
		organizationID,
	)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberByID(
		ctx,
		sqlc.GetPhoneNumberByIDParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) Update(
	ctx context.Context,
	organizationID, id uuid.UUID,
	req UpdateRequest,
) (sqlc.PhoneNumber, error) {
	return r.queries.UpdatePhoneNumber(
		ctx,
		sqlc.UpdatePhoneNumberParams{
			ID:             id,
			OrganizationID: organizationID,
			VoiceEnabled:   req.VoiceEnabled,
		},
	)
}

func (r *Repository) SetTrunk(
	ctx context.Context,
	organizationID, id, trunkID uuid.UUID,
) (sqlc.PhoneNumber, error) {
	return r.queries.SetBYOCPhoneNumberTrunk(
		ctx,
		sqlc.SetBYOCPhoneNumberTrunkParams{
			ID:             id,
			OrganizationID: organizationID,
			TrunkID:        &trunkID,
		},
	)
}

func (r *Repository) Release(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.PhoneNumber, error) {
	return r.queries.ReleaseBYOCPhoneNumber(
		ctx,
		sqlc.ReleaseBYOCPhoneNumberParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}
