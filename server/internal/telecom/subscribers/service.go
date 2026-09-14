package subscribers

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/hasher"
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
) (sqlc.Subscriber, error) {
	if err := validOrg(organizationID); err != nil {
		return sqlc.Subscriber{}, err
	}

	if req.SIPDomainID == uuid.Nil {
		return sqlc.Subscriber{}, apperror.NewBadRequest("sip_domain_id is required")
	}

	var err error
	req.Username, err = normalizeUsername(req.Username)
	if err != nil {
		return sqlc.Subscriber{}, err
	}

	if err := validatePassword(req.Password); err != nil {
		return sqlc.Subscriber{}, err
	}

	req.DisplayName, err = normalizeDisplayName(req.DisplayName)
	if err != nil {
		return sqlc.Subscriber{}, err
	}

	domain, err := s.repo.Domain(ctx, organizationID, req.SIPDomainID)
	if err != nil {
		return sqlc.Subscriber{}, readError(err, "SIP domain not found")
	}

	hashes := hasher.ComputeHA1(req.Username, domain.Domain, req.Password)
	result, err := s.repo.Create(ctx, organizationID, req, hashes)
	return result, writeError(err, "create subscriber")
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Subscriber, error) {
	if err := validOrg(organizationID); err != nil {
		return nil, err
	}

	result, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list subscribers", err)
	}

	return result, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscriber, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.Subscriber{}, err
	}

	result, err := s.repo.Get(ctx, organizationID, id)
	return result, readError(err, "subscriber not found")
}

func (s *Service) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req UpdateRequest,
) (sqlc.Subscriber, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.Subscriber{}, err
	}

	if req.DisplayName == nil {
		return sqlc.Subscriber{}, apperror.NewBadRequest("at least one field is required")
	}

	name, err := normalizeDisplayName(req.DisplayName)
	if err != nil {
		return sqlc.Subscriber{}, err
	}

	result, err := s.repo.Update(ctx, organizationID, id, name)
	return result, writeError(err, "subscriber not found")
}

func (s *Service) Rotate(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req RotateCredentialsRequest,
) (sqlc.Subscriber, error) {
	subscriber, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return sqlc.Subscriber{}, err
	}

	if err := validatePassword(req.Password); err != nil {
		return sqlc.Subscriber{}, err
	}

	hashes := hasher.ComputeHA1(subscriber.Username, subscriber.Domain, req.Password)
	result, err := s.repo.Rotate(ctx, organizationID, id, hashes)
	return result, writeError(err, "subscriber not found")
}

func (s *Service) Delete(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}

	return writeError(s.repo.Disable(ctx, organizationID, id), "disable subscriber")
}

func validIDs(organizationID, id uuid.UUID) error {
	if err := validOrg(organizationID); err != nil {
		return err
	}

	return validID(id)
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
		return apperror.NewConflict("subscriber already exists")
	}

	return readError(err, message)
}
