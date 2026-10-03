package agents

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/ai/providers"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo      *Repository
	providers *providers.Service
}

func NewService(repo *Repository, providerServices ...*providers.Service) *Service {
	var providerService *providers.Service
	if len(providerServices) > 0 {
		providerService = providerServices[0]
	}
	return &Service{
		repo:      repo,
		providers: providerService,
	}
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.VoiceAgent, error) {
	if organizationID == uuid.Nil {
		return sqlc.VoiceAgent{}, apperror.NewBadRequest("organization_id is required")
	}
	normalized, err := normalizeCreate(req)
	if err != nil {
		return sqlc.VoiceAgent{}, err
	}
	agent, err := s.repo.Create(ctx, organizationID, normalized)
	if err != nil {
		return agent, writeError(err, "create voice agent")
	}
	if err := s.applyBindings(ctx, organizationID, agent.ID, normalized.Bindings, false); err != nil {
		return sqlc.VoiceAgent{}, err
	}
	if len(normalized.Bindings) > 0 {
		return s.Get(ctx, organizationID, agent.ID)
	}
	return agent, nil
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.VoiceAgent, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	agents, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list voice agents", err)
	}
	return agents, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.VoiceAgent, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.VoiceAgent{}, err
	}
	agent, err := s.repo.Get(ctx, organizationID, id)
	return agent, readError(err, "voice agent not found")
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.VoiceAgent, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.VoiceAgent{}, err
	}
	normalized, err := normalizeUpdate(req)
	if err != nil {
		return sqlc.VoiceAgent{}, err
	}
	agent, err := s.repo.Update(ctx, organizationID, id, normalized)
	if err != nil {
		return agent, writeError(err, "update voice agent")
	}
	if normalized.Bindings != nil {
		if err := s.applyBindings(ctx, organizationID, id, *normalized.Bindings, true); err != nil {
			return sqlc.VoiceAgent{}, err
		}
		return s.Get(ctx, organizationID, id)
	}
	return agent, nil
}

func (s *Service) applyBindings(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
	bindings []ProviderBindingRequest,
	replace bool,
) error {
	if len(bindings) == 0 && !replace {
		return nil
	}
	if s.providers == nil {
		return apperror.NewServiceUnavailable("AI provider service is unavailable", nil)
	}
	if replace {
		existing, err := s.providers.ListBindings(ctx, organizationID, agentID)
		if err != nil {
			return err
		}
		requested := make(map[string]struct{}, len(bindings))
		for _, binding := range bindings {
			requested[binding.Role] = struct{}{}
		}
		for _, binding := range existing {
			if _, ok := requested[binding.Role]; ok {
				continue
			}
			if err := s.providers.DeleteBinding(
				ctx,
				organizationID,
				agentID,
				binding.Role,
			); err != nil {
				return err
			}
		}
	}
	for _, binding := range bindings {
		_, err := s.providers.UpsertBinding(
			ctx,
			organizationID,
			agentID,
			binding.Role,
			providers.UpsertBindingRequest{
				Provider:     binding.Provider,
				CredentialID: binding.IntegrationID,
				Config:       binding.Config,
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Disable(ctx context.Context, organizationID, id uuid.UUID) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	return writeError(s.repo.Disable(ctx, organizationID, id), "disable voice agent")
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
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	if conflict(err) {
		return apperror.NewConflict("voice agent already exists")
	}
	return apperror.NewInternal(message, err)
}

func conflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
