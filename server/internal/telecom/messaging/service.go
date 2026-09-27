package messaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	if repo == nil {
		panic("messaging: repository is required")
	}
	return &Service{repo: repo}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	idempotencyKey string,
	req CreateRequest,
) (sqlc.Message, error) {
	if organizationID == uuid.Nil {
		return sqlc.Message{}, apperror.NewBadRequest("organization context required")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || len(idempotencyKey) > 255 {
		return sqlc.Message{}, apperror.NewBadRequest("Idempotency-Key is required and must not exceed 255 characters")
	}
	normalized, err := normalizeCreateRequest(req)
	if err != nil {
		return sqlc.Message{}, err
	}
	hash, err := requestHash(normalized)
	if err != nil {
		return sqlc.Message{}, apperror.NewInternal("hash message request", err)
	}

	value, err := s.repo.CreateOutbound(ctx, organizationID, idempotencyKey, hash, normalized)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Message{}, apperror.NewInternal("create message", err)
	}

	existing, readErr := s.repo.GetByIdempotencyKey(ctx, organizationID, idempotencyKey)
	if readErr != nil {
		return sqlc.Message{}, messageReadError(readErr, "message not found")
	}
	if existing.RequestHash == nil || *existing.RequestHash != hash {
		return sqlc.Message{}, apperror.NewConflict("idempotency key was used with another message request")
	}
	return existing, nil
}

func (s *Service) CreateInbound(ctx context.Context, req InboundRequest) (sqlc.Message, error) {
	normalized, err := normalizeInboundRequest(req)
	if err != nil {
		return sqlc.Message{}, err
	}
	value, err := s.repo.CreateInbound(ctx, normalized)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Message{}, apperror.NewInternal("create inbound message", err)
	}
	existing, readErr := s.repo.GetByProviderID(ctx, normalized.CarrierConnectionID, normalized.ProviderMessageID)
	if readErr != nil {
		return sqlc.Message{}, messageReadError(readErr, "message not found")
	}
	return existing, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Message, error) {
	if organizationID == uuid.Nil {
		return sqlc.Message{}, apperror.NewBadRequest("organization context required")
	}
	if id == uuid.Nil {
		return sqlc.Message{}, apperror.NewBadRequest("message id is required")
	}
	value, err := s.repo.Get(ctx, organizationID, id)
	return value, messageReadError(err, "message not found")
}

// GetByProviderID resolves the carrier-scoped identity used by inbound delivery
// receipts. It is intentionally not exposed as an organization API lookup.
func (s *Service) GetByProviderID(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	providerMessageID string,
) (sqlc.Message, error) {
	value, err := s.repo.GetByProviderID(ctx, carrierConnectionID, providerMessageID)
	return value, messageReadError(err, "message not found")
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID, req ListRequest) ([]sqlc.Message, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization context required")
	}
	req = normalizeListRequest(req)
	if err := validateListRequest(req); err != nil {
		return nil, err
	}
	values, err := s.repo.List(ctx, organizationID, req)
	if err != nil {
		return nil, apperror.NewInternal("list messages", err)
	}
	return values, nil
}

func (s *Service) SetProviderAttribution(
	ctx context.Context,
	organizationID, id, carrierConnectionID uuid.UUID,
) (sqlc.Message, error) {
	if organizationID == uuid.Nil || id == uuid.Nil || carrierConnectionID == uuid.Nil {
		return sqlc.Message{}, apperror.NewBadRequest(
			"organization_id, message_id, and carrier_connection_id are required",
		)
	}
	value, err := s.repo.SetProviderAttribution(ctx, organizationID, id, carrierConnectionID)
	return value, messageWriteError(err, "attribute message provider")
}

func (s *Service) MarkSubmitted(
	ctx context.Context,
	organizationID, id uuid.UUID,
	providerMessageID string,
) (sqlc.Message, error) {
	providerMessageID = strings.TrimSpace(providerMessageID)
	if providerMessageID == "" || len(providerMessageID) > 255 {
		return sqlc.Message{}, apperror.NewBadRequest(
			"provider_message_id must be between 1 and 255 characters",
		)
	}
	value, err := s.repo.MarkSubmitted(ctx, organizationID, id, providerMessageID)
	return value, messageWriteError(err, "submit message")
}

func (s *Service) MarkSent(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Message, error) {
	value, err := s.repo.MarkSent(ctx, organizationID, id)
	return value, messageWriteError(err, "mark message sent")
}

func (s *Service) MarkDelivered(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Message, error) {
	value, err := s.repo.MarkDelivered(ctx, organizationID, id)
	return value, messageWriteError(err, "mark message delivered")
}

func (s *Service) MarkUndelivered(
	ctx context.Context,
	organizationID, id uuid.UUID,
	failure Failure,
) (sqlc.Message, error) {
	value, err := s.repo.MarkUndelivered(ctx, organizationID, id, failure)
	return value, messageWriteError(err, "mark message undelivered")
}

func (s *Service) MarkFailed(
	ctx context.Context,
	organizationID, id uuid.UUID,
	failure Failure,
) (sqlc.Message, error) {
	value, err := s.repo.MarkFailed(ctx, organizationID, id, failure)
	return value, messageWriteError(err, "mark message failed")
}

func requestHash(req CreateRequest) (string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func messageWriteError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewConflict(message + " is not valid in the current message state")
	}
	if err != nil {
		return apperror.NewInternal(message, err)
	}
	return nil
}

func messageReadError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	if err != nil {
		return apperror.NewInternal("get message", err)
	}
	return nil
}
