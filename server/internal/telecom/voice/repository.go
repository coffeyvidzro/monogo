package voice

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, organizationID uuid.UUID, req CreateApplicationRequest) (sqlc.VoiceApplication, error) {
	return r.queries.CreateVoiceApplication(ctx, sqlc.CreateVoiceApplicationParams{
		OrganizationID:     organizationID,
		Name:               req.Name,
		RingTimeoutSeconds: req.RingTimeoutSeconds,
		CallerID:           req.CallerID,
		VoiceUrl:           req.VoiceURL,
		CallbackUrl:        req.CallbackURL,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.VoiceApplication, error) {
	return r.queries.ListVoiceApplicationsByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.VoiceApplication, error) {
	return r.queries.GetVoiceApplicationByID(ctx, sqlc.GetVoiceApplicationByIDParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateApplicationRequest) (sqlc.VoiceApplication, error) {
	return r.queries.UpdateVoiceApplication(ctx, sqlc.UpdateVoiceApplicationParams{
		Name:               req.Name,
		RingTimeoutSeconds: req.RingTimeoutSeconds,
		CallerID:           req.CallerID,
		VoiceUrl:           req.VoiceURL,
		CallbackUrl:        req.CallbackURL,
		ID:                 id,
		OrganizationID:     organizationID,
	})
}

func (r *Repository) Disable(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DisableVoiceApplication(ctx, sqlc.DisableVoiceApplicationParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) CreateBinding(ctx context.Context, organizationID, applicationID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceBinding, error) {
	return r.queries.CreateVoiceBinding(ctx, sqlc.CreateVoiceBindingParams{
		VoiceApplicationID: applicationID,
		PhoneNumberID:      req.PhoneNumberID,
		SipDomainID:        req.SIPDomainID,
		SubscriberID:       req.SubscriberID,
		OrganizationID:     organizationID,
	})
}

func (r *Repository) ListBindings(ctx context.Context, organizationID, applicationID uuid.UUID) ([]sqlc.VoiceBinding, error) {
	return r.queries.ListVoiceBindingsByApplicationID(ctx, sqlc.ListVoiceBindingsByApplicationIDParams{
		VoiceApplicationID: applicationID,
		OrganizationID:     organizationID,
	})
}

func (r *Repository) DeleteBinding(ctx context.Context, organizationID, applicationID, id uuid.UUID) error {
	return r.queries.DeleteVoiceBinding(ctx, sqlc.DeleteVoiceBindingParams{
		ID:                 id,
		VoiceApplicationID: applicationID,
		OrganizationID:     organizationID,
	})
}
