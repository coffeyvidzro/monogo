package numbers

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateBYOC(ctx context.Context, organizationID uuid.UUID, req CreateBYOCRequest) (sqlc.PhoneNumber, error) {
	if err := validateOrganization(organizationID); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if err := normalizeBYOC(&req); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	row, err := s.repo.CreateBYOC(ctx, organizationID, req)
	return row, writeError(err)
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.PhoneNumber, error) {
	if err := validateOrganization(organizationID); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list numbers", err)
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	row, err := s.repo.Get(ctx, organizationID, id)
	return row, readError(err)
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.PhoneNumber, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if err := validateUpdate(req); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	row, err := s.repo.Update(ctx, organizationID, id, req)
	return row, writeError(err)
}

func (s *Service) SetBYOCConnection(ctx context.Context, organizationID, id uuid.UUID, req SetCarrierConnectionRequest) (sqlc.PhoneNumber, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if req.CarrierConnectionID == uuid.Nil {
		return sqlc.PhoneNumber{}, apperror.NewBadRequest("carrier_connection_id is required")
	}
	row, err := s.repo.SetBYOCConnection(ctx, organizationID, id, req.CarrierConnectionID)
	return row, writeError(err)
}

func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	number, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if number.ProvisioningMode != "byoc" {
		return apperror.NewConflict("managed number release requires provider deprovisioning")
	}
	_, err = s.repo.ReleaseBYOC(ctx, organizationID, id)
	return writeError(err)
}

func readError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("number not found")
	}
	return apperror.NewInternal("get number", err)
}

func writeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("number or carrier connection not found")
	}
	var dbError *pgconn.PgError
	if errors.As(err, &dbError) {
		switch dbError.Code {
		case "23505":
			return apperror.NewConflict("number already exists")
		case "23503", "23514", "23502":
			return apperror.NewBadRequest("number or carrier connection is invalid")
		}
	}
	return apperror.NewInternal("update number", err)
}
