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

func (r *Repository) CreateBYOC(ctx context.Context, organizationID uuid.UUID, req CreateBYOCRequest) (sqlc.PhoneNumber, error) {
	return r.queries.CreateBYOCPhoneNumber(ctx, sqlc.CreateBYOCPhoneNumberParams{
		OrganizationID:      organizationID,
		Number:              req.Number,
		CountryCode:         req.CountryCode,
		CarrierConnectionID: req.CarrierConnectionID,
		VoiceEnabled:        req.VoiceEnabled,
		SmsEnabled:          req.SmsEnabled,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.PhoneNumber, error) {
	return r.queries.ListPhoneNumbersByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberByID(ctx, sqlc.GetPhoneNumberByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.PhoneNumber, error) {
	return r.queries.UpdatePhoneNumber(ctx, sqlc.UpdatePhoneNumberParams{
		ID:             id,
		OrganizationID: organizationID,
		VoiceEnabled:   req.VoiceEnabled,
		SmsEnabled:     req.SmsEnabled,
	})
}

func (r *Repository) SetBYOCConnection(ctx context.Context, organizationID, id, connectionID uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.SetBYOCPhoneNumberCarrierConnection(ctx, sqlc.SetBYOCPhoneNumberCarrierConnectionParams{
		ID:                  id,
		OrganizationID:      organizationID,
		CarrierConnectionID: &connectionID,
	})
}

func (r *Repository) ReleaseBYOC(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.ReleaseBYOCPhoneNumber(ctx, sqlc.ReleaseBYOCPhoneNumberParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}
