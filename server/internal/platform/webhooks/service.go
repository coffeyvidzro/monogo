package webhooks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
	db   *pgxpool.Pool
}

func NewService(repository *Repository, db ...*pgxpool.Pool) *Service {
	service := &Service{repo: repository}
	if len(db) > 0 {
		service.db = db[0]
	}
	return service
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.WebhookEndpoint, []byte, error) {
	if err := validOrg(organizationID); err != nil {
		return sqlc.WebhookEndpoint{}, nil, err
	}
	if err := normalizeCreate(&req); err != nil {
		return sqlc.WebhookEndpoint{}, nil, err
	}
	secret, err := newSigningSecret()
	if err != nil {
		return sqlc.WebhookEndpoint{}, nil, apperror.NewInternal("generate webhook secret", err)
	}
	v, err := s.repo.Create(ctx, organizationID, req, secret)
	return v, secret, readErr(err, "create webhook")
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.WebhookEndpoint, error) {
	if err := validOrg(organizationID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, organizationID)
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.WebhookEndpoint, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.WebhookEndpoint{}, err
	}
	v, err := s.repo.Get(ctx, organizationID, id)
	return v, readErr(err, "webhook not found")
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.WebhookEndpoint, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.WebhookEndpoint{}, err
	}
	if err := normalizeUpdate(&req); err != nil {
		return sqlc.WebhookEndpoint{}, err
	}
	v, err := s.repo.Update(ctx, organizationID, id, req)
	return v, readErr(err, "webhook not found")
}

func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	if err := validIDs(organizationID, id); err != nil {
		return err
	}
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	if err := s.repo.Disable(ctx, organizationID, id); err != nil {
		return readErr(err, "disable webhook")
	}
	return readErr(s.repo.Cancel(ctx, organizationID, id), "cancel webhook deliveries")
}

func (s *Service) RotateSecret(ctx context.Context, organizationID, id uuid.UUID) (sqlc.WebhookEndpoint, []byte, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.WebhookEndpoint{}, nil, err
	}
	secret, err := newSigningSecret()
	if err != nil {
		return sqlc.WebhookEndpoint{}, nil, apperror.NewInternal("generate webhook secret", err)
	}
	v, err := s.repo.RotateSecret(ctx, organizationID, id, secret)
	return v, secret, readErr(err, "webhook not found")
}

func (s *Service) ListDeliveries(ctx context.Context, organizationID, id uuid.UUID, limit, offset int32) ([]sqlc.WebhookDelivery, error) {
	if err := validIDs(organizationID, id); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, apperror.NewBadRequest("limit must be between 1 and 100")
	}
	if offset < 0 {
		return nil, apperror.NewBadRequest("offset must not be negative")
	}
	return s.repo.ListDeliveries(ctx, organizationID, id, limit, offset)
}

func (s *Service) GetDelivery(ctx context.Context, organizationID, endpoint, id uuid.UUID) (sqlc.WebhookDelivery, error) {
	if err := validIDs(organizationID, endpoint); err != nil {
		return sqlc.WebhookDelivery{}, err
	}
	v, err := s.repo.GetDelivery(ctx, organizationID, id)
	if err = readErr(err, "webhook delivery not found"); err != nil {
		return sqlc.WebhookDelivery{}, err
	}
	if v.EndpointID != endpoint {
		return sqlc.WebhookDelivery{}, apperror.NewNotFound("webhook delivery not found")
	}
	return v, nil
}

func (s *Service) Retry(ctx context.Context, organizationID, endpoint, id uuid.UUID) (sqlc.WebhookDelivery, error) {
	if _, err := s.GetDelivery(ctx, organizationID, endpoint, id); err != nil {
		return sqlc.WebhookDelivery{}, err
	}
	v, err := s.repo.Retry(ctx, organizationID, id)
	return v, readErr(err, "webhook delivery is not retryable")
}

func (s *Service) Ingest(ctx context.Context, event InboundEvent) error {
	if err := validateInboundEvent(event); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("webhook database is required for event ingestion")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin webhook event transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := s.repo.WithTx(tx)
	if _, err := repo.CreateEvent(ctx, event); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("create webhook event: %w", err)
		}
		existing, getErr := repo.GetEvent(ctx, event.OrganizationID, event.ID)
		if getErr != nil {
			return fmt.Errorf("get existing webhook event: %w", getErr)
		}
		if existing.EventType != event.EventType || existing.ObjectType != event.ObjectType {
			return fmt.Errorf("webhook event %s conflicts with existing event metadata", event.ID)
		}
	}

	if _, err := repo.CreateDeliveriesForEvent(ctx, event.ID, time.Now().UTC()); err != nil {
		return fmt.Errorf("create webhook deliveries: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit webhook event transaction: %w", err)
	}
	return nil
}

func readErr(err error, msg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(msg)
	}
	return apperror.NewInternal(msg, err)
}

func (s *Service) Test(ctx context.Context, organizationID, id uuid.UUID) (int, error) {
	v, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return 0, err
	}
	if !v.Enabled {
		return 0, apperror.NewBadRequest("webhook is disabled")
	}
	status, err := sendTest(ctx, v)
	if err != nil {
		return 0, apperror.NewInternal("send webhook test", err)
	}
	return status, nil
}
