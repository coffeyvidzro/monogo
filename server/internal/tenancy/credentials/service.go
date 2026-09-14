package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/security/authn"
)

var ErrInvalidInput = errors.New("invalid credential input")

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, input CreateInput) (CreatedCredential, error) {
	if err := ValidateCreate(input); err != nil {
		return CreatedCredential{}, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	token, prefix, hash, err := GenerateToken()
	if err != nil {
		return CreatedCredential{}, err
	}
	row, err := s.repository.Create(ctx, input, hash, prefix)
	if err != nil {
		return CreatedCredential{}, err
	}
	credential, err := organizationTokenToCredential(row)
	if err != nil {
		return CreatedCredential{}, err
	}
	return CreatedCredential{Credential: credential, Token: token}, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (Credential, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Credential{}, fmt.Errorf("%w: organization and credential ids are required", ErrInvalidInput)
	}
	row, err := s.repository.GetByID(ctx, organizationID, id)
	if err != nil {
		return Credential{}, err
	}
	return organizationTokenToCredential(row)
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]Credential, error) {
	if organizationID == uuid.Nil {
		return nil, fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	rows, err := s.repository.List(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	credentials := make([]Credential, 0, len(rows))
	for _, row := range rows {
		credential, err := listRowToCredential(row)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}
	return credentials, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Credential, error) {
	if token == "" {
		return Credential{}, ErrInvalidToken
	}
	if _, err := ParseTokenPrefix(token); err != nil {
		return Credential{}, ErrInvalidToken
	}
	row, err := s.repository.GetByTokenHash(ctx, HashToken(token))
	if err != nil {
		return Credential{}, ErrInvalidToken
	}
	credential, err := organizationTokenToCredential(row)
	if err != nil {
		return Credential{}, ErrInvalidToken
	}
	return credential, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput, actorID uuid.UUID) (Credential, error) {
	if err := ValidateUpdate(input); err != nil {
		return Credential{}, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	if actorID == uuid.Nil {
		return Credential{}, fmt.Errorf("%w: actor id is required", ErrInvalidInput)
	}
	row, err := s.repository.Update(ctx, input, actorID)
	if err != nil {
		return Credential{}, err
	}
	return organizationTokenToCredential(row)
}

func (s *Service) Disable(ctx context.Context, organizationID, id, actorID uuid.UUID) error {
	if organizationID == uuid.Nil || id == uuid.Nil || actorID == uuid.Nil {
		return fmt.Errorf("%w: organization, credential, and actor ids are required", ErrInvalidInput)
	}
	return s.repository.Disable(ctx, organizationID, id, actorID)
}

func (s *Service) Touch(ctx context.Context, id uuid.UUID, ip netip.Addr) error {
	if id == uuid.Nil || !ip.IsValid() {
		return fmt.Errorf("%w: credential id and valid ip are required", ErrInvalidInput)
	}
	return s.repository.Touch(ctx, id, ip)
}

func organizationTokenToCredential(row sqlc.OrganizationToken) (Credential, error) {
	var scopes []string
	if err := json.Unmarshal(row.Scopes, &scopes); err != nil {
		return Credential{}, fmt.Errorf("decode credential scopes: %w", err)
	}
	if err := ValidateScopes(scopes); err != nil {
		return Credential{}, fmt.Errorf("invalid credential scopes: %w", err)
	}
	return Credential{
		ID: row.ID, OrganizationID: row.OrganizationID, CreatedBy: row.CreatedBy,
		Name: row.Name, Description: row.Description, TokenPrefix: row.TokenPrefix,
		Scopes: scopes, LastUsedAt: pgconv.TimestamptzToTimePtr(row.LastUsedAt),
		ExpiresAt: pgconv.TimestamptzToTimePtr(row.ExpiresAt), DisabledAt: pgconv.TimestamptzToTimePtr(row.DisabledAt),
		CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt),
		LastUsedIP: pgconv.String(row.LastUsedIp),
	}, nil
}

func listRowToCredential(row sqlc.ListOrganizationTokensByOrganizationIDRow) (Credential, error) {
	var scopes []string
	if err := json.Unmarshal(row.Scopes, &scopes); err != nil {
		return Credential{}, fmt.Errorf("decode credential scopes: %w", err)
	}
	if err := ValidateScopes(scopes); err != nil {
		return Credential{}, fmt.Errorf("invalid credential scopes: %w", err)
	}
	return Credential{
		ID: row.ID, Name: row.Name, Description: row.Description, TokenPrefix: row.TokenPrefix,
		Scopes: scopes, LastUsedAt: pgconv.TimestamptzToTimePtr(row.LastUsedAt),
		ExpiresAt: pgconv.TimestamptzToTimePtr(row.ExpiresAt), CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt),
	}, nil
}

// ResolveOrganizationToken resolves an organization token for the authentication layer.
func (s *Service) ResolveOrganizationToken(ctx context.Context, token string) (authn.OrganizationToken, error) {
	credential, err := s.Authenticate(ctx, token)
	if err != nil {
		return authn.OrganizationToken{}, err
	}

	return authn.OrganizationToken{
		ID:             credential.ID,
		OrganizationID: credential.OrganizationID,
		Scopes:         credential.Scopes,
	}, nil
}
