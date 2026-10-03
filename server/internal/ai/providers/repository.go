package providers

import (
	"context"
	"encoding/json"
	"time"

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

func (r *Repository) CreateCredential(
	ctx context.Context,
	id uuid.UUID,
	organizationID uuid.UUID,
	provider string,
	name string,
	ciphertext string,
) (Credential, error) {
	row, err := r.queries.CreateAIProviderCredential(
		ctx,
		sqlc.CreateAIProviderCredentialParams{
			ID:               id,
			OrganizationID:   organizationID,
			Provider:         provider,
			Name:             name,
			SecretCiphertext: ciphertext,
		},
	)
	if err != nil {
		return Credential{}, err
	}

	return credentialFromRow(row), nil
}

func (r *Repository) ListCredentials(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Credential, error) {
	rows, err := r.queries.ListAIProviderCredentials(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	result := make([]Credential, 0, len(rows))
	for _, row := range rows {
		result = append(result, credentialFromRow(row))
	}

	return result, nil
}

func (r *Repository) GetCredentialCiphertext(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Credential, string, error) {
	row, err := r.queries.GetAIProviderCredential(
		ctx,
		sqlc.GetAIProviderCredentialParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		return Credential{}, "", err
	}

	return credentialFromRow(row), row.SecretCiphertext, nil
}

func (r *Repository) RotateCredential(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	ciphertext string,
) (Credential, error) {
	row, err := r.queries.RotateAIProviderCredential(
		ctx,
		sqlc.RotateAIProviderCredentialParams{
			ID:               id,
			OrganizationID:   organizationID,
			SecretCiphertext: ciphertext,
		},
	)
	if err != nil {
		return Credential{}, err
	}

	return credentialFromRow(row), nil
}

func (r *Repository) UpdateVerification(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	state string,
	failureCode *string,
) (Credential, error) {
	row, err := r.queries.UpdateAIProviderCredentialVerification(
		ctx,
		sqlc.UpdateAIProviderCredentialVerificationParams{
			ConnectionState: state,
			FailureCode:     failureCode,
			ID:              id,
			OrganizationID:  organizationID,
		},
	)
	if err != nil {
		return Credential{}, err
	}
	return credentialFromRow(row), nil
}

func (r *Repository) ListVoiceAgentIDs(
	ctx context.Context,
	organizationID uuid.UUID,
	credentialID uuid.UUID,
) ([]uuid.UUID, error) {
	return r.queries.ListVoiceAgentIDsByAIProviderCredential(
		ctx,
		sqlc.ListVoiceAgentIDsByAIProviderCredentialParams{
			OrganizationID: organizationID,
			CredentialID:   credentialID,
		},
	)
}

func (r *Repository) DeleteCredential(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	return r.queries.DeleteAIProviderCredential(
		ctx,
		sqlc.DeleteAIProviderCredentialParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) UpsertBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
	role string,
	req UpsertBindingRequest,
) (Binding, error) {
	config := req.Config
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}

	row, err := r.queries.UpsertVoiceAgentProviderBinding(
		ctx,
		sqlc.UpsertVoiceAgentProviderBindingParams{
			OrganizationID: organizationID,
			VoiceAgentID:   agentID,
			Role:           role,
			Provider:       req.Provider,
			CredentialID:   req.CredentialID,
			Config:         config,
		},
	)
	if err != nil {
		return Binding{}, err
	}

	return Binding{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		VoiceAgentID:   row.VoiceAgentID,
		Role:           row.Role,
		Provider:       row.Provider,
		CredentialID:   row.CredentialID,
		Config:         json.RawMessage(row.Config),
	}, nil
}

func (r *Repository) ListBindings(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) ([]Binding, error) {
	rows, err := r.queries.ListVoiceAgentProviderBindings(
		ctx,
		sqlc.ListVoiceAgentProviderBindingsParams{
			OrganizationID: organizationID,
			VoiceAgentID:   agentID,
		},
	)
	if err != nil {
		return nil, err
	}

	result := make([]Binding, 0, len(rows))
	for _, row := range rows {
		result = append(result, Binding{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			VoiceAgentID:   row.VoiceAgentID,
			Role:           row.Role,
			Provider:       row.Provider,
			CredentialID:   row.CredentialID,
			Config:         json.RawMessage(row.Config),
		})
	}

	return result, nil
}

func (r *Repository) DeleteBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
	role string,
) error {
	return r.queries.DeleteVoiceAgentProviderBinding(
		ctx,
		sqlc.DeleteVoiceAgentProviderBindingParams{
			OrganizationID: organizationID,
			VoiceAgentID:   agentID,
			Role:           role,
		},
	)
}

func (r *Repository) ResolveBindings(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) ([]resolvedRow, error) {
	rows, err := r.queries.ResolveVoiceAgentProviderBindings(
		ctx,
		sqlc.ResolveVoiceAgentProviderBindingsParams{
			OrganizationID: organizationID,
			VoiceAgentID:   agentID,
		},
	)
	if err != nil {
		return nil, err
	}

	result := make([]resolvedRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, resolvedRow{
			Role:             row.Role,
			Provider:         row.Provider,
			CredentialID:     row.CredentialID,
			Config:           json.RawMessage(row.Config),
			SecretCiphertext: row.SecretCiphertext,
		})
	}

	return result, nil
}

func credentialFromRow(row sqlc.AiProviderCredential) Credential {
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		value := pgconv.TimestamptzToTime(row.VerifiedAt)
		verifiedAt = &value
	}
	return Credential{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Provider:        row.Provider,
		Name:            row.Name,
		ConnectionState: row.ConnectionState,
		VerifiedAt:      verifiedAt,
		FailureCode:     row.FailureCode,
		CreatedAt:       pgconv.TimestamptzToTime(row.CreatedAt),
		RotatedAt:       pgconv.TimestamptzToTime(row.RotatedAt),
		UpdatedAt:       pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

type resolvedRow struct {
	Role             string
	Provider         string
	CredentialID     uuid.UUID
	Config           json.RawMessage
	SecretCiphertext string
}
