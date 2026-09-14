package voice

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateApplicationRequest) (sqlc.VoiceApplication, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	name, err := normalizeName(req.Name)
	if err != nil {
		return sqlc.VoiceApplication{}, err
	}
	if err := validateRingTimeout(req.RingTimeoutSeconds); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	req.Name = name
	if req.CallerID, err = normalizeOptionalString(req.CallerID, "caller_id"); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	if req.VoiceURL, err = normalizeOptionalURL(req.VoiceURL, "voice_url"); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	if req.CallbackURL, err = normalizeOptionalURL(req.CallbackURL, "callback_url"); err != nil {
		return sqlc.VoiceApplication{}, err
	}

	app, err := s.repo.Create(ctx, organizationID, req)
	return app, translateWriteError(err, "create voice application")
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.VoiceApplication, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, organizationID)
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.VoiceApplication, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	app, err := s.repo.Get(ctx, organizationID, id)
	return app, translateReadError(err, "voice application not found")
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateApplicationRequest) (sqlc.VoiceApplication, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	var err error
	if req.Name != nil {
		name, err := normalizeName(*req.Name)
		if err != nil {
			return sqlc.VoiceApplication{}, err
		}
		req.Name = &name
	}
	if err := validateRingTimeout(req.RingTimeoutSeconds); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	if req.CallerID, err = normalizeOptionalString(req.CallerID, "caller_id"); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	if req.VoiceURL, err = normalizeOptionalURL(req.VoiceURL, "voice_url"); err != nil {
		return sqlc.VoiceApplication{}, err
	}
	if req.CallbackURL, err = normalizeOptionalURL(req.CallbackURL, "callback_url"); err != nil {
		return sqlc.VoiceApplication{}, err
	}

	app, err := s.repo.Update(ctx, organizationID, id, req)
	return app, translateReadError(err, "voice application not found")
}

func (s *Service) Disable(ctx context.Context, organizationID, id uuid.UUID) error {
	if err := validateIDs(organizationID, id); err != nil {
		return err
	}
	return translateWriteError(s.repo.Disable(ctx, organizationID, id), "disable voice application")
}

func (s *Service) CreateBinding(ctx context.Context, organizationID, applicationID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceBinding, error) {
	if err := validateIDs(organizationID, applicationID); err != nil {
		return sqlc.VoiceBinding{}, err
	}
	if err := validateBindingTarget(req); err != nil {
		return sqlc.VoiceBinding{}, err
	}
	binding, err := s.repo.CreateBinding(ctx, organizationID, applicationID, req)
	return binding, translateWriteError(err, "create voice binding")
}

func (s *Service) ListBindings(ctx context.Context, organizationID, applicationID uuid.UUID) ([]sqlc.VoiceBinding, error) {
	if err := validateIDs(organizationID, applicationID); err != nil {
		return nil, err
	}
	return s.repo.ListBindings(ctx, organizationID, applicationID)
}

func (s *Service) DeleteBinding(ctx context.Context, organizationID, applicationID, id uuid.UUID) error {
	if err := validateIDs(organizationID, applicationID); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("binding id is required")
	}
	return translateWriteError(s.repo.DeleteBinding(ctx, organizationID, applicationID, id), "delete voice binding")
}

func validateIDs(organizationID, applicationID uuid.UUID) error {
	if err := validateOrganizationID(organizationID); err != nil {
		return err
	}
	return validateApplicationID(applicationID)
}

func translateReadError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(notFoundMessage)
	}
	return apperror.NewInternal(notFoundMessage, err)
}

func translateWriteError(err error, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}
