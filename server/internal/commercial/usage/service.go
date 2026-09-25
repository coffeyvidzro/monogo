package usage

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Record is internal-only. Usage recording neither charges a wallet nor
// establishes a billable amount; billing authorization is a separate workflow.
func (s *Service) Record(ctx context.Context, req RecordRequest) (sqlc.UsageEvent, error) {
	if err := validateRecord(&req); err != nil {
		return sqlc.UsageEvent{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.UsageEvent{}, apperror.NewServiceUnavailable("usage persistence is not configured", nil)
	}
	event, err := s.repo.Record(ctx, req)
	if err == nil {
		return event, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewInternal("record usage observation", err)
	}
	// ON CONFLICT DO NOTHING returns no row for an existing key. Verify
	// every material field, not just the idempotency key, before replaying.
	existing, err := s.repo.ByKey(ctx, req.OrganizationID, req.IdempotencyKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewConflict("meter unavailable for usage recording")
	}
	if err != nil {
		return sqlc.UsageEvent{}, apperror.NewInternal("resolve usage observation", err)
	}
	matched, err := s.repo.MatchingByKey(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewConflict("usage idempotency key already used for another observation")
	}
	if err != nil {
		return sqlc.UsageEvent{}, apperror.NewInternal("verify usage observation replay", err)
	}
	if matched.ID != existing.ID {
		return sqlc.UsageEvent{}, apperror.NewConflict("usage observation replay mismatch")
	}
	return existing, nil
}

func (s *Service) Get(ctx context.Context, organizationID, eventID uuid.UUID) (sqlc.UsageEvent, error) {
	if organizationID == uuid.Nil || eventID == uuid.Nil {
		return sqlc.UsageEvent{}, apperror.NewBadRequest("organization and usage event are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.UsageEvent{}, apperror.NewServiceUnavailable("usage persistence is not configured", nil)
	}
	event, err := s.repo.Get(ctx, organizationID, eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.UsageEvent{}, apperror.NewNotFound("usage event not found")
	}
	if err != nil {
		return sqlc.UsageEvent{}, apperror.NewInternal("get usage event", err)
	}
	return event, nil
}
