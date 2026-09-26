package agents

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.VoiceAgent, error) {
	return r.queries.CreateVoiceAgent(ctx, sqlc.CreateVoiceAgentParams{
		OrganizationID: organizationID,
		Name: req.Name,
		Engine: req.Engine,
		Instructions: req.Instructions,
		Voice: req.Voice,
		Language: req.Language,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.VoiceAgent, error) {
	return r.queries.ListVoiceAgentsByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.VoiceAgent, error) {
	return r.queries.GetVoiceAgentByID(ctx, sqlc.GetVoiceAgentByIDParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.VoiceAgent, error) {
	return r.queries.UpdateVoiceAgent(ctx, sqlc.UpdateVoiceAgentParams{
		Name: req.Name,
		Engine: req.Engine,
		Instructions: req.Instructions,
		Voice: req.Voice,
		Language: req.Language,
		ID: id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Disable(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DisableVoiceAgent(ctx, sqlc.DisableVoiceAgentParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) ResolveByApplication(ctx context.Context, organizationID, applicationID uuid.UUID) (sqlc.VoiceAgent, error) {
	return r.queries.GetVoiceAgentByApplicationID(ctx, sqlc.GetVoiceAgentByApplicationIDParams{
		OrganizationID: organizationID,
		VoiceApplicationID: applicationID,
	})
}
