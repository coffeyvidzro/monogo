package scim

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Principal struct {
	TokenID        uuid.UUID
	OrganizationID uuid.UUID
}

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}
func (s *Service) CreateToken(ctx context.Context, organizationID uuid.UUID, req CreateTokenRequest) (CreatedToken, error) {
	if organizationID == uuid.Nil {
		return CreatedToken{}, apperror.NewBadRequest("organization_id is required")
	}
	req, err := normalizeCreateToken(req, time.Now().UTC())
	if err != nil {
		return CreatedToken{}, err
	}
	secret, prefix, hash, err := generateToken()
	if err != nil {
		return CreatedToken{}, apperror.NewInternal("generate SCIM token", err)
	}
	value, err := s.repo.Create(ctx, organizationID, req, prefix, hash)
	if err != nil {
		return CreatedToken{}, apperror.NewInternal("create SCIM token", err)
	}
	return CreatedToken{
		Token:  value,
		Secret: secret,
	}, nil
}
func (s *Service) ListTokens(ctx context.Context, organizationID uuid.UUID) ([]Token, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list SCIM tokens", err)
	}
	return values, nil
}
func (s *Service) RevokeToken(ctx context.Context, organizationID, id uuid.UUID) (Token, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Token{}, apperror.NewBadRequest("organization and token ids are required")
	}
	value, err := s.repo.Revoke(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, apperror.NewNotFound("SCIM token not found")
	}
	if err != nil {
		return Token{}, apperror.NewInternal("revoke SCIM token", err)
	}
	return value, nil
}
func (s *Service) Authenticate(ctx context.Context, secret string) (Principal, error) {
	if len(secret) <= len(tokenPrefix) || secret[:len(tokenPrefix)] != tokenPrefix {
		return Principal{}, apperror.NewUnauthorized("invalid SCIM token")
	}
	row, err := s.repo.Authenticate(ctx, hashToken(secret))
	if err != nil {
		return Principal{}, apperror.NewUnauthorized("invalid SCIM token")
	}
	if err := s.repo.Touch(ctx, row.OrganizationID, row.ID); err != nil {
		return Principal{}, apperror.NewInternal("update SCIM token usage", err)
	}
	return Principal{
		TokenID:        row.ID,
		OrganizationID: row.OrganizationID,
	}, nil
}
