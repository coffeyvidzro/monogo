package agents

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateBindingRequest struct {
	VoiceApplicationID uuid.UUID `json:"voice_application_id"`
}

type BindingResponse struct {
	ID                 uuid.UUID `json:"id"`
	OrganizationID     uuid.UUID `json:"organization_id"`
	VoiceAgentID       uuid.UUID `json:"voice_agent_id"`
	VoiceApplicationID uuid.UUID `json:"voice_application_id"`
	CreatedAt          time.Time `json:"created_at"`
}

func (r *Repository) CreateBinding(ctx context.Context, organizationID, agentID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceAgentBinding, error) {
	return r.queries.CreateVoiceAgentBinding(ctx, sqlc.CreateVoiceAgentBindingParams{
		OrganizationID:     organizationID,
		VoiceAgentID:       agentID,
		VoiceApplicationID: req.VoiceApplicationID,
	})
}

func (r *Repository) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentBinding, error) {
	return r.queries.ListVoiceAgentBindingsByAgentID(ctx, sqlc.ListVoiceAgentBindingsByAgentIDParams{
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) DeleteBinding(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	return r.queries.DeleteVoiceAgentBinding(ctx, sqlc.DeleteVoiceAgentBindingParams{
		ID: id, OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}

func (s *Service) CreateBinding(ctx context.Context, organizationID, agentID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceAgentBinding, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentBinding{}, err
	}
	if req.VoiceApplicationID == uuid.Nil {
		return sqlc.VoiceAgentBinding{}, apperror.NewBadRequest("voice_application_id is required")
	}
	binding, err := s.repo.CreateBinding(ctx, organizationID, agentID, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.VoiceAgentBinding{}, apperror.NewNotFound("voice agent or voice application not found")
	}
	if conflict(err) {
		return sqlc.VoiceAgentBinding{}, apperror.NewConflict("voice application already has a voice agent")
	}
	if err != nil {
		return sqlc.VoiceAgentBinding{}, apperror.NewInternal("create voice agent binding", err)
	}
	return binding, nil
}

func (s *Service) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentBinding, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, organizationID, agentID); err != nil {
		return nil, err
	}
	bindings, err := s.repo.ListBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal("list voice agent bindings", err)
	}
	return bindings, nil
}

func (s *Service) DeleteBinding(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	if err := validateIDs(organizationID, agentID); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("binding id is required")
	}
	return writeError(s.repo.DeleteBinding(ctx, organizationID, agentID, id), "delete voice agent binding")
}

func bindingResponse(binding sqlc.VoiceAgentBinding) BindingResponse {
	return BindingResponse{
		ID:                 binding.ID,
		OrganizationID:     binding.OrganizationID,
		VoiceAgentID:       binding.VoiceAgentID,
		VoiceApplicationID: binding.VoiceApplicationID,
		CreatedAt:          pgconv.TimestamptzToTime(binding.CreatedAt),
	}
}
