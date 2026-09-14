package sip_domains

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (sqlc.SipDomain, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.SipDomain{}, err
	}

	domain, err := normalizeDomain(req.Domain)
	if err != nil {
		return sqlc.SipDomain{}, err
	}

	result, err := s.repo.Create(ctx, organizationID, domain)
	return result, writeError(err, "create SIP domain")
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.SipDomain, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}

	domains, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list SIP domains", err)
	}

	return domains, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.SipDomain, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.SipDomain{}, err
	}

	if err := validateDomainID(id); err != nil {
		return sqlc.SipDomain{}, err
	}

	domain, err := s.repo.Get(ctx, organizationID, id)
	return domain, readError(err, "SIP domain not found")
}

func (s *Service) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req UpdateRequest,
) (sqlc.SipDomain, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.SipDomain{}, err
	}

	if err := validateDomainID(id); err != nil {
		return sqlc.SipDomain{}, err
	}

	if req.Domain == nil {
		return sqlc.SipDomain{}, apperror.NewBadRequest("at least one field is required")
	}

	domain, err := normalizeDomain(*req.Domain)
	if err != nil {
		return sqlc.SipDomain{}, err
	}

	result, err := s.repo.Update(ctx, organizationID, id, &domain)
	return result, writeError(err, "SIP domain not found")
}

func (s *Service) Delete(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}

	if err := s.repo.Disable(ctx, organizationID, id); err != nil {
		return writeError(err, "disable SIP domain")
	}

	return nil
}

func readError(err error, message string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}

	return apperror.NewInternal(message, err)
}

func writeError(err error, message string) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.NewConflict("SIP domain already exists")
	}

	return readError(err, message)
}
