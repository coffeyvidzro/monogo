package tools

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo   *Repository
	cipher *encryption.Cipher
}

func NewService(repo *Repository, cipher *encryption.Cipher) *Service {
	return &Service{repo: repo, cipher: cipher}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID, agentID uuid.UUID,
	req CreateRequest,
) (sqlc.VoiceAgentTool, string, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentTool{}, "", err
	}
	normalized, err := normalizeCreate(req)
	if err != nil {
		return sqlc.VoiceAgentTool{}, "", err
	}

	if normalized.Type == TypeBuiltin {
		tool, createErr := s.repo.Create(ctx, organizationID, agentID, normalized)
		if conflict(createErr) {
			return sqlc.VoiceAgentTool{}, "", apperror.NewConflict("voice agent tool already exists")
		}
		if createErr != nil {
			return sqlc.VoiceAgentTool{}, "", dbError(createErr, "create voice agent tool")
		}
		return tool, "", nil
	}
	if s.cipher == nil {
		return sqlc.VoiceAgentTool{}, "", apperror.NewServiceUnavailable(
			"voice agent tool signing is unavailable",
			nil,
		)
	}

	toolID := uuid.New()
	secret, err := newSigningSecret()
	if err != nil {
		return sqlc.VoiceAgentTool{}, "", apperror.NewInternal(
			"generate voice agent tool signing secret",
			err,
		)
	}
	ciphertext, err := s.cipher.EncryptForScope(
		signingScope(organizationID, agentID, toolID),
		secret,
	)
	if err != nil {
		return sqlc.VoiceAgentTool{}, "", apperror.NewInternal(
			"encrypt voice agent tool signing secret",
			err,
		)
	}
	tool, err := s.repo.CreateWebhook(
		ctx,
		organizationID,
		agentID,
		toolID,
		normalized,
		ciphertext,
	)
	if conflict(err) {
		return sqlc.VoiceAgentTool{}, "", apperror.NewConflict("voice agent tool already exists")
	}
	if err != nil {
		return sqlc.VoiceAgentTool{}, "", dbError(err, "create voice agent tool")
	}
	return tool, secret, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID, agentID, id uuid.UUID,
) (sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentTool{}, err
	}
	if id == uuid.Nil {
		return sqlc.VoiceAgentTool{}, apperror.NewBadRequest("tool id is required")
	}
	tool, err := s.repo.Get(ctx, organizationID, agentID, id)
	return tool, dbError(err, "voice agent tool not found")
}

func (s *Service) SigningSecret(
	ctx context.Context,
	organizationID, agentID, toolID uuid.UUID,
) (string, error) {
	if s.cipher == nil {
		return "", apperror.NewServiceUnavailable("voice agent tool signing is unavailable", nil)
	}
	ciphertext, err := s.repo.GetSigningSecret(ctx, organizationID, agentID, toolID)
	if err != nil {
		return "", dbError(err, "voice agent tool signing secret not found")
	}
	secret, err := s.cipher.DecryptForScope(
		signingScope(organizationID, agentID, toolID),
		ciphertext,
	)
	if err != nil {
		return "", apperror.NewInternal("decrypt voice agent tool signing secret", err)
	}
	return secret, nil
}

func (s *Service) RotateSigningSecret(
	ctx context.Context,
	organizationID, agentID, toolID uuid.UUID,
) (string, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return "", err
	}
	if toolID == uuid.Nil {
		return "", apperror.NewBadRequest("tool id is required")
	}
	if s.cipher == nil {
		return "", apperror.NewServiceUnavailable("voice agent tool signing is unavailable", nil)
	}
	secret, err := newSigningSecret()
	if err != nil {
		return "", apperror.NewInternal("generate voice agent tool signing secret", err)
	}
	ciphertext, err := s.cipher.EncryptForScope(
		signingScope(organizationID, agentID, toolID),
		secret,
	)
	if err != nil {
		return "", apperror.NewInternal("encrypt voice agent tool signing secret", err)
	}
	rows, err := s.repo.RotateSigningSecret(
		ctx,
		organizationID,
		agentID,
		toolID,
		ciphertext,
	)
	if err != nil {
		return "", apperror.NewInternal("rotate voice agent tool signing secret", err)
	}
	if rows != 1 {
		return "", apperror.NewNotFound("webhook voice agent tool not found")
	}
	return secret, nil
}

func (s *Service) List(
	ctx context.Context,
	organizationID, agentID uuid.UUID,
) ([]sqlc.VoiceAgentTool, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return nil, err
	}
	items, err := s.repo.List(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal("list voice agent tools", err)
	}
	return items, nil
}

func (s *Service) Update(
	ctx context.Context,
	organizationID, agentID, id uuid.UUID,
	req UpdateRequest,
) (sqlc.VoiceAgentTool, error) {
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
	if normalized.Name != nil || normalized.EndpointURL != nil {
		current, readErr := s.Get(ctx, organizationID, agentID, id)
		if readErr != nil {
			return sqlc.VoiceAgentTool{}, readErr
		}
		if normalized.Name != nil && current.Type == TypeBuiltin {
			switch *normalized.Name {
			case BuiltinHangupCall, BuiltinTransferCall, BuiltinHoldCall, BuiltinResumeCall, BuiltinSendDTMF:
			default:
				return sqlc.VoiceAgentTool{}, apperror.NewBadRequest("unsupported builtin tool name")
			}
		}
		if normalized.EndpointURL != nil {
			if err := validateEndpoint(current.Type, normalized.EndpointURL); err != nil {
				return sqlc.VoiceAgentTool{}, err
			}
		}
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
