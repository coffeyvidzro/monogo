package sso

import (
	"context"
	"errors"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo   *Repository
	cipher *encryption.Cipher
}

func NewService(repo *Repository, cipher *encryption.Cipher) *Service {
	return &Service{repo: repo, cipher: cipher}
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (Connection, error) {
	if organizationID == uuid.Nil {
		return Connection{}, apperror.NewBadRequest("organization_id is required")
	}
	if err := normalizeCreate(&req); err != nil {
		return Connection{}, err
	}
	id := uuid.New()
	ciphertext, err := s.encryptSecret(organizationID, id, req.Secret)
	if err != nil {
		return Connection{}, err
	}
	value, err := s.repo.Create(ctx, id, organizationID, req, ciphertext)
	if conflict(err) {
		return Connection{}, apperror.NewConflict("an SSO connection with this name already exists")
	}
	if err != nil {
		return Connection{}, apperror.NewInternal("create SSO connection", err)
	}
	return value, nil
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]Connection, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list SSO connections", err)
	}
	return values, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (Connection, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Connection{}, apperror.NewBadRequest("organization and SSO connection ids are required")
	}
	value, _, err := s.repo.Get(ctx, organizationID, id)
	if err != nil {
		return Connection{}, databaseError(err, "SSO connection not found")
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (Connection, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Connection{}, apperror.NewBadRequest("organization and SSO connection ids are required")
	}
	if err := normalizeUpdate(&req); err != nil {
		return Connection{}, err
	}
	if _, _, err := s.repo.Get(ctx, organizationID, id); err != nil {
		return Connection{}, databaseError(err, "SSO connection not found")
	}
	ciphertext, err := s.encryptSecret(organizationID, id, req.Secret)
	if err != nil {
		return Connection{}, err
	}
	value, err := s.repo.Update(ctx, organizationID, id, req, ciphertext)
	if conflict(err) {
		return Connection{}, apperror.NewConflict("an SSO connection with this name already exists")
	}
	if err != nil {
		return Connection{}, databaseError(err, "SSO connection not found")
	}
	return value, nil
}

func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return apperror.NewBadRequest("organization and SSO connection ids are required")
	}
	if _, _, err := s.repo.Get(ctx, organizationID, id); err != nil {
		return databaseError(err, "SSO connection not found")
	}
	if err := s.repo.Delete(ctx, organizationID, id); err != nil {
		return apperror.NewInternal("delete SSO connection", err)
	}
	return nil
}

func (s *Service) Resolve(ctx context.Context, organizationID, id uuid.UUID) (Connection, string, error) {
	value, ciphertext, err := s.repo.Get(ctx, organizationID, id)
	if err != nil {
		return Connection{}, "", databaseError(err, "SSO connection not found")
	}
	if value.Status != StatusActive {
		return Connection{}, "", apperror.NewForbidden("SSO connection is disabled")
	}
	if ciphertext == nil {
		return value, "", nil
	}
	if s.cipher == nil {
		return Connection{}, "", apperror.NewServiceUnavailable("SSO credential encryption is unavailable", nil)
	}
	secret, err := s.cipher.DecryptForScope(credentialScope(organizationID, id), *ciphertext)
	if err != nil {
		return Connection{}, "", apperror.NewInternal("decrypt SSO credential", err)
	}
	return value, secret, nil
}

func (s *Service) encryptSecret(organizationID, id uuid.UUID, secret *string) (*string, error) {
	if secret == nil {
		return nil, nil
	}
	if s.cipher == nil {
		return nil, apperror.NewServiceUnavailable("SSO credential encryption is unavailable", nil)
	}
	ciphertext, err := s.cipher.EncryptForScope(credentialScope(organizationID, id), *secret)
	if err != nil {
		return nil, apperror.NewInternal("encrypt SSO credential", err)
	}
	return &ciphertext, nil
}

func credentialScope(organizationID, connectionID uuid.UUID) string {
	return fmt.Sprintf("sso-connection:%s:%s", organizationID, connectionID)
}

func databaseError(err error, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}

func conflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
