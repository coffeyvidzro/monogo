package voice_agents

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct { repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.VoiceAgent, error) {
	if organizationID == uuid.Nil { return sqlc.VoiceAgent{}, apperror.NewBadRequest("organization_id is required") }
	normalized, err := normalizeCreate(req)
	if err != nil { return sqlc.VoiceAgent{}, err }
	agent, err := s.repo.Create(ctx, organizationID, normalized)
	return agent, writeError(err, "create voice agent")
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.VoiceAgent, error) {
	if organizationID == uuid.Nil { return nil, apperror.NewBadRequest("organization_id is required") }
	agents, err := s.repo.List(ctx, organizationID)
	if err != nil { return nil, apperror.NewInternal("list voice agents", err) }
	return agents, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.VoiceAgent, error) {
	if err := validateIDs(organizationID, id); err != nil { return sqlc.VoiceAgent{}, err }
	agent, err := s.repo.Get(ctx, organizationID, id)
	return agent, readError(err, "voice agent not found")
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.VoiceAgent, error) {
	if err := validateIDs(organizationID, id); err != nil { return sqlc.VoiceAgent{}, err }
	normalized, err := normalizeUpdate(req)
	if err != nil { return sqlc.VoiceAgent{}, err }
	agent, err := s.repo.Update(ctx, organizationID, id, normalized)
	return agent, writeError(err, "update voice agent")
}

func (s *Service) Disable(ctx context.Context, organizationID, id uuid.UUID) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil { return err }
	return writeError(s.repo.Disable(ctx, organizationID, id), "disable voice agent")
}

func (s *Service) CreateBinding(ctx context.Context, organizationID, agentID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceAgentBinding, error) {
	if err := validateIDs(organizationID, agentID); err != nil { return sqlc.VoiceAgentBinding{}, err }
	if err := validateBinding(req); err != nil { return sqlc.VoiceAgentBinding{}, err }
	binding, err := s.repo.CreateBinding(ctx, organizationID, agentID, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.VoiceAgentBinding{}, apperror.NewNotFound("voice agent or voice application not found")
	}
	if conflict(err) {
		return sqlc.VoiceAgentBinding{}, apperror.NewConflict("voice application already has a voice agent")
	}
	if err != nil { return sqlc.VoiceAgentBinding{}, apperror.NewInternal("create voice agent binding", err) }
	return binding, nil
}

func (s *Service) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentBinding, error) {
	if err := validateIDs(organizationID, agentID); err != nil { return nil, err }
	if _, err := s.Get(ctx, organizationID, agentID); err != nil { return nil, err }
	bindings, err := s.repo.ListBindings(ctx, organizationID, agentID)
	if err != nil { return nil, apperror.NewInternal("list voice agent bindings", err) }
	return bindings, nil
}

func (s *Service) DeleteBinding(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	if err := validateIDs(organizationID, agentID); err != nil { return err }
	if id == uuid.Nil { return apperror.NewBadRequest("binding id is required") }
	return writeError(s.repo.DeleteBinding(ctx, organizationID, agentID, id), "delete voice agent binding")
}

func (s *Service) ResolveByApplication(ctx context.Context, organizationID, applicationID uuid.UUID) (sqlc.VoiceAgent, error) {
	if organizationID == uuid.Nil { return sqlc.VoiceAgent{}, apperror.NewBadRequest("organization_id is required") }
	if applicationID == uuid.Nil { return sqlc.VoiceAgent{}, apperror.NewBadRequest("voice application id is required") }
	agent, err := s.repo.ResolveByApplication(ctx, organizationID, applicationID)
	return agent, readError(err, "voice agent not found")
}

func readError(err error, message string) error {
	if err == nil { return nil }
	if errors.Is(err, pgx.ErrNoRows) { return apperror.NewNotFound(message) }
	return apperror.NewInternal(message, err)
}

func writeError(err error, message string) error {
	if err == nil { return nil }
	if errors.Is(err, pgx.ErrNoRows) { return apperror.NewNotFound(message) }
	if conflict(err) { return apperror.NewConflict("voice agent already exists") }
	return apperror.NewInternal(message, err)
}

func conflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
