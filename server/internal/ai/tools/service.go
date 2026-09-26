package tools

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, organizationID, agentID uuid.UUID, req CreateRequest) (sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	normalized, err := normalizeCreate(req)
	if err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	tool, err := s.repo.Create(ctx, organizationID, agentID, normalized)
	if conflict(err) {
		return sqlc.VoiceAgentTool{}, apperror.NewConflict("voice agent tool already exists")
	}
	if err != nil {
		return sqlc.VoiceAgentTool{}, dbError(err, "create voice agent tool")
	}
	return tool, nil
}

func (s *Service) List(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return nil, err
	}
	items, err := s.repo.List(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal("list voice agent tools", err)
	}
	return items, nil
}

func (s *Service) Update(ctx context.Context, organizationID, agentID, id uuid.UUID, req UpdateRequest) (sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	if id == uuid.Nil {
		return sqlc.VoiceAgentTool{}, apperror.NewBadRequest("tool id is required")
	}
	normalized, err := normalizeUpdate(req)
	if err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	tool, err := s.repo.Update(ctx, organizationID, agentID, id, normalized)
	if conflict(err) {
		return sqlc.VoiceAgentTool{}, apperror.NewConflict("voice agent tool already exists")
	}
	return tool, dbError(err, "voice agent tool not found")
}

func (s *Service) Delete(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	if err := validateIDs(organizationID, agentID); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("tool id is required")
	}
	return dbError(s.repo.Delete(ctx, organizationID, agentID, id), "delete voice agent tool")
}

func validateIDs(organizationID, agentID uuid.UUID) error {
	if organizationID == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	if agentID == uuid.Nil {
		return apperror.NewBadRequest("voice agent id is required")
	}
	return nil
}

func dbError(err error, message string) error {
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
