package charges

import (
	"bytes"
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultListLimit int32 = 50
	maxListLimit     int32 = 200
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	if repo == nil {
		panic("billing charges: repository is required")
	}

	return &Service{
		repo: repo,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (sqlc.Charge, error) {
	if err := normalizeCreate(&req); err != nil {
		return sqlc.Charge{}, err
	}

	charge, err := s.repo.Create(
		ctx,
		req,
	)
	if err == nil {
		return charge, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		existing, readErr := s.repo.ByIdempotencyKey(
			ctx,
			req.OrganizationID,
			req.IdempotencyKey,
		)
		if readErr != nil {
			return sqlc.Charge{}, chargeReadError(readErr)
		}

		if !sameRequest(existing, req) {
			return sqlc.Charge{}, apperror.NewConflict(
				"idempotency key was used with another charge request",
			)
		}

		return existing, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" {
		_, readErr := s.repo.ByResource(
			ctx,
			req.OrganizationID,
			req.ResourceType,
			req.ResourceID,
		)
		if readErr == nil {
			return sqlc.Charge{}, apperror.NewConflict(
				"resource already has a charge",
			)
		}
	}

	return sqlc.Charge{}, apperror.NewInternal(
		"create billing charge",
		err,
	)
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Charge, error) {
	if organizationID == uuid.Nil ||
		id == uuid.Nil {
		return sqlc.Charge{}, apperror.NewBadRequest(
			"organization and charge are required",
		)
	}

	charge, err := s.repo.Get(
		ctx,
		organizationID,
		id,
	)

	return charge, chargeReadError(err)
}

func (s *Service) ByResource(
	ctx context.Context,
	organizationID uuid.UUID,
	resourceType string,
	resourceID uuid.UUID,
) (sqlc.Charge, error) {
	if organizationID == uuid.Nil ||
		resourceID == uuid.Nil {
		return sqlc.Charge{}, apperror.NewBadRequest(
			"organization and resource are required",
		)
	}

	resourceType, err := normalizeResourceType(
		resourceType,
	)
	if err != nil {
		return sqlc.Charge{}, err
	}

	charge, err := s.repo.ByResource(
		ctx,
		organizationID,
		resourceType,
		resourceID,
	)

	return charge, chargeReadError(err)
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
	req ListRequest,
) ([]sqlc.Charge, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest(
			"organization context required",
		)
	}

	if err := normalizeListRequest(&req); err != nil {
		return nil, err
	}

	values, err := s.repo.List(
		ctx,
		organizationID,
		req,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list billing charges",
			err,
		)
	}

	return values, nil
}

func sameRequest(
	value sqlc.Charge,
	req CreateRequest,
) bool {
	return value.WalletID == req.WalletID &&
		value.ResourceType == req.ResourceType &&
		value.ResourceID == req.ResourceID &&
		value.ChargingMode == req.ChargingMode &&
		value.Currency == req.Currency &&
		value.RequestHash == req.RequestHash &&
		bytes.Equal(
			value.PricingSnapshot,
			req.PricingSnapshot,
		)
}

func chargeReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(
			"charge not found",
		)
	}
	if err != nil {
		return apperror.NewInternal(
			"get billing charge",
			err,
		)
	}

	return nil
}
