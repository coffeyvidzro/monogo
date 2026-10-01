package providers

import (
	"context"
	"encoding/json"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) CreateCredential(ctx context.Context, id, organizationID uuid.UUID, provider, name, ciphertext string) (Credential, error) {
	row, err := r.queries.CreateAIProviderCredential(ctx, sqlc.CreateAIProviderCredentialParams{
		ID: id, OrganizationID: organizationID, Provider: provider, Name: name, SecretCiphertext: ciphertext,
	})
	if err != nil {
		return Credential{}, err
	}
	return credentialFromRow(row.ID, row.OrganizationID, row.Provider, row.Name, row.CreatedAt, row.RotatedAt, row.UpdatedAt), nil
}

func (r *Repository) ListCredentials(ctx context.Context, organizationID uuid.UUID) ([]Credential, error) {
	rows, err := r.queries.ListAIProviderCredentials(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]Credential, 0, len(rows))
	for _, row := range rows {
		out = append(out, credentialFromRow(row.ID, row.OrganizationID, row.Provider, row.Name, row.CreatedAt, row.RotatedAt, row.UpdatedAt))
	}
	return out, nil
}

func (r *Repository) GetCredentialCiphertext(ctx context.Context, organizationID, id uuid.UUID) (Credential, string, error) {
	row, err := r.queries.GetAIProviderCredential(ctx, sqlc.GetAIProviderCredentialParams{ID: id, OrganizationID: organizationID})
	if err != nil {
		return Credential{}, "", err
	}
	return credentialFromRow(row.ID, row.OrganizationID, row.Provider, row.Name, row.CreatedAt, row.RotatedAt, row.UpdatedAt), row.SecretCiphertext, nil
}

func (r *Repository) RotateCredential(ctx context.Context, organizationID, id uuid.UUID, ciphertext string) (Credential, error) {
	row, err := r.queries.RotateAIProviderCredential(ctx, sqlc.RotateAIProviderCredentialParams{
		ID: id, OrganizationID: organizationID, SecretCiphertext: ciphertext,
	})
	if err != nil {
		return Credential{}, err
	}
	return credentialFromRow(row.ID, row.OrganizationID, row.Provider, row.Name, row.CreatedAt, row.RotatedAt, row.UpdatedAt), nil
}

func (r *Repository) DeleteCredential(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DeleteAIProviderCredential(ctx, sqlc.DeleteAIProviderCredentialParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) UpsertBinding(ctx context.Context, organizationID, agentID uuid.UUID, role string, req UpsertBindingRequest) (Binding, error) {
	config := req.Config
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	row, err := r.queries.UpsertVoiceAgentProviderBinding(ctx, sqlc.UpsertVoiceAgentProviderBindingParams{
		OrganizationID: organizationID, VoiceAgentID: agentID, Role: role, Provider: req.Provider,
		CredentialID: req.CredentialID, Config: config,
	})
	if err != nil {
		return Binding{}, err
	}
	return Binding{
		ID: row.ID, OrganizationID: row.OrganizationID, VoiceAgentID: row.VoiceAgentID, Role: row.Role,
		Provider: row.Provider, CredentialID: row.CredentialID, Config: json.RawMessage(row.Config),
	}, nil
}

func (r *Repository) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]Binding, error) {
	rows, err := r.queries.ListVoiceAgentProviderBindings(ctx, sqlc.ListVoiceAgentProviderBindingsParams{
		OrganizationID: organizationID, VoiceAgentID: agentID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Binding, 0, len(rows))
	for _, row := range rows {
		out = append(out, Binding{
			ID: row.ID, OrganizationID: row.OrganizationID, VoiceAgentID: row.VoiceAgentID, Role: row.Role,
			Provider: row.Provider, CredentialID: row.CredentialID, Config: json.RawMessage(row.Config),
		})
	}
	return out, nil
}

func (r *Repository) DeleteBinding(ctx context.Context, organizationID, agentID uuid.UUID, role string) error {
	return r.queries.DeleteVoiceAgentProviderBinding(ctx, sqlc.DeleteVoiceAgentProviderBindingParams{
		OrganizationID: organizationID, VoiceAgentID: agentID, Role: role,
	})
}

func (r *Repository) ResolveBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]resolvedRow, error) {
	rows, err := r.queries.ResolveVoiceAgentProviderBindings(ctx, sqlc.ResolveVoiceAgentProviderBindingsParams{
		OrganizationID: organizationID, VoiceAgentID: agentID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]resolvedRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, resolvedRow{
			Role: row.Role, Provider: row.Provider, CredentialID: row.CredentialID,
			Config: json.RawMessage(row.Config), SecretCiphertext: row.SecretCiphertext,
		})
	}
	return out, nil
}

func credentialFromRow(id, organizationID uuid.UUID, provider, name string, createdAt, rotatedAt, updatedAt pgtype.Timestamptz) Credential {
	return Credential{
		ID: id, OrganizationID: organizationID, Provider: provider, Name: name,
		CreatedAt: pgconv.TimestamptzToTime(createdAt),
		RotatedAt: pgconv.TimestamptzToTime(rotatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(updatedAt),
	}
}

type resolvedRow struct {
	Role             string
	Provider         string
	CredentialID     uuid.UUID
	Config           json.RawMessage
	SecretCiphertext string
}
