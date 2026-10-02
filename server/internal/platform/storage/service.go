package storage

import (
	"context"
	"errors"
	"fmt"

	s3integration "github.com/coffeyvidzro/monogo/internal/integrations/s3"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo                  *Repository
	cipher                *encryption.Cipher
	allowPrivateEndpoints bool
}

func NewService(
	repo *Repository,
	cipher *encryption.Cipher,
	allowPrivateEndpoints ...bool,
) *Service {
	allowPrivate := len(allowPrivateEndpoints) > 0 && allowPrivateEndpoints[0]
	return &Service{
		repo:                  repo,
		cipher:                cipher,
		allowPrivateEndpoints: allowPrivate,
	}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (Integration, error) {
	if organizationID == uuid.Nil {
		return Integration{}, apperror.NewBadRequest("organization_id is required")
	}
	if err := normalizeCreateWithPolicy(&req, s.allowPrivateEndpoints); err != nil {
		return Integration{}, err
	}
	if s.cipher == nil {
		return Integration{}, apperror.NewServiceUnavailable(
			"storage credential encryption is unavailable",
			nil,
		)
	}

	id := uuid.New()
	ciphertext, err := s.cipher.EncryptForScope(
		credentialScope(organizationID, id),
		req.SecretAccessKey,
	)
	if err != nil {
		return Integration{}, apperror.NewInternal(
			"encrypt storage credential",
			err,
		)
	}

	value, err := s.repo.Create(ctx, id, organizationID, req, ciphertext)
	if conflict(err) {
		return Integration{}, apperror.NewConflict(
			"an active recording storage integration already exists",
		)
	}
	if err != nil {
		return Integration{}, databaseError(err, "create storage integration")
	}
	return value, nil
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Integration, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list storage integrations", err)
	}
	return values, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Integration, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Integration{}, apperror.NewBadRequest(
			"organization and storage integration ids are required",
		)
	}
	value, _, err := s.repo.Get(ctx, organizationID, id)
	if err != nil {
		return Integration{}, databaseError(err, "storage integration not found")
	}
	return value, nil
}

func (s *Service) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req UpdateRequest,
) (Integration, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Integration{}, apperror.NewBadRequest(
			"organization and storage integration ids are required",
		)
	}
	if err := normalizeUpdate(&req); err != nil {
		return Integration{}, err
	}
	if _, _, err := s.repo.Get(ctx, organizationID, id); err != nil {
		return Integration{}, databaseError(err, "storage integration not found")
	}

	var ciphertext *string
	if req.SecretAccessKey != nil {
		if s.cipher == nil {
			return Integration{}, apperror.NewServiceUnavailable(
				"storage credential encryption is unavailable",
				nil,
			)
		}
		value, err := s.cipher.EncryptForScope(
			credentialScope(organizationID, id),
			*req.SecretAccessKey,
		)
		if err != nil {
			return Integration{}, apperror.NewInternal(
				"encrypt storage credential",
				err,
			)
		}
		ciphertext = &value
	}

	value, err := s.repo.Update(ctx, organizationID, id, req, ciphertext)
	if conflict(err) {
		return Integration{}, apperror.NewConflict(
			"another active recording storage integration already exists",
		)
	}
	if err != nil {
		return Integration{}, databaseError(err, "storage integration not found")
	}
	return value, nil
}

func (s *Service) Delete(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and storage integration ids are required",
		)
	}
	_, err := s.repo.Disable(ctx, organizationID, id)
	return databaseError(err, "storage integration not found")
}

func (s *Service) ResolveRecording(
	ctx context.Context,
	organizationID uuid.UUID,
) (ResolvedIntegration, error) {
	if organizationID == uuid.Nil {
		return ResolvedIntegration{}, apperror.NewBadRequest(
			"organization_id is required",
		)
	}
	value, ciphertext, err := s.repo.GetActiveRecording(ctx, organizationID)
	if err != nil {
		return ResolvedIntegration{}, err
	}
	return s.resolve(value, ciphertext)
}

func (s *Service) ResolveByID(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (ResolvedIntegration, error) {
	value, ciphertext, err := s.repo.Get(ctx, organizationID, id)
	if err != nil {
		return ResolvedIntegration{}, err
	}
	return s.resolve(value, ciphertext)
}

func (s *Service) Test(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	resolved, err := s.ResolveByID(ctx, organizationID, id)
	if err != nil {
		return databaseError(err, "storage integration not found")
	}
	client, err := s3integration.New(ctx, s3integration.Config{
		Endpoint:              resolved.EndpointURL,
		Region:                resolved.Region,
		Bucket:                resolved.Bucket,
		AccessKey:             resolved.AccessKeyID,
		SecretKey:             resolved.SecretAccessKey,
		UsePathStyle:          resolved.UsePathStyle,
		AllowPrivateEndpoints: resolved.AllowPrivateEndpoint,
	})
	if err != nil {
		return apperror.NewBadRequest(err.Error())
	}
	if err := client.Test(ctx); err != nil {
		return apperror.NewBadRequest(
			fmt.Sprintf("storage connection test failed: %v", err),
		)
	}
	return nil
}

func (s *Service) resolve(
	value Integration,
	ciphertext string,
) (ResolvedIntegration, error) {
	if s.cipher == nil {
		return ResolvedIntegration{}, apperror.NewServiceUnavailable(
			"storage credential encryption is unavailable",
			nil,
		)
	}
	secret, err := s.cipher.DecryptForScope(
		credentialScope(value.OrganizationID, value.ID),
		ciphertext,
	)
	if err != nil {
		return ResolvedIntegration{}, apperror.NewInternal(
			"decrypt storage credential",
			err,
		)
	}
	return ResolvedIntegration{
		Integration:          value,
		SecretAccessKey:      secret,
		AllowPrivateEndpoint: s.allowPrivateEndpoints,
	}, nil
}

func credentialScope(organizationID, integrationID uuid.UUID) string {
	return fmt.Sprintf(
		"storage-integration:%s:%s",
		organizationID,
		integrationID,
	)
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
