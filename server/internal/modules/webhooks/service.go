package webhooks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct {
	repo *Repository
	db   *pgxpool.Pool
}

func NewService(r *Repository, db ...*pgxpool.Pool) *Service {
	service := &Service{repo: r}
	if len(db) > 0 {
		service.db = db[0]
	}
	return service
}
func (s *Service) Create(c context.Context, org uuid.UUID, req CreateRequest) (sqlc.WebhookEndpoint, []byte, error) {
	if err := validOrg(org); err != nil {
		return sqlc.WebhookEndpoint{}, nil, err
	}
	if err := normalizeCreate(&req); err != nil {
		return sqlc.WebhookEndpoint{}, nil, err
	}
	secret, err := newSigningSecret()
	if err != nil {
		return sqlc.WebhookEndpoint{}, nil, apperror.NewInternal("generate webhook secret", err)
	}
	v, err := s.repo.Create(c, org, req, secret)
	return v, secret, writeErr(err, "create webhook")
}
func (s *Service) List(c context.Context, org uuid.UUID) ([]sqlc.WebhookEndpoint, error) {
	if err := validOrg(org); err != nil {
		return nil, err
	}
	return s.repo.List(c, org)
}
func (s *Service) Get(c context.Context, org, id uuid.UUID) (sqlc.WebhookEndpoint, error) {
	if err := validIDs(org, id); err != nil {
		return sqlc.WebhookEndpoint{}, err
	}
	v, e := s.repo.Get(c, org, id)
	return v, readErr(e, "webhook not found")
}
func (s *Service) Update(c context.Context, org, id uuid.UUID, req UpdateRequest) (sqlc.WebhookEndpoint, error) {
	if err := validIDs(org, id); err != nil {
		return sqlc.WebhookEndpoint{}, err
	}
	if err := normalizeUpdate(&req); err != nil {
		return sqlc.WebhookEndpoint{}, err
	}
	v, e := s.repo.Update(c, org, id, req)
	return v, readErr(e, "webhook not found")
}
func (s *Service) Delete(c context.Context, org, id uuid.UUID) error {
	if err := validIDs(org, id); err != nil {
		return err
	}
	if _, err := s.Get(c, org, id); err != nil {
		return err
	}
	if err := s.repo.Disable(c, org, id); err != nil {
		return writeErr(err, "disable webhook")
	}
	return writeErr(s.repo.Cancel(c, org, id), "cancel webhook deliveries")
}
func (s *Service) RotateSecret(c context.Context, org, id uuid.UUID) (sqlc.WebhookEndpoint, []byte, error) {
	if err := validIDs(org, id); err != nil {
		return sqlc.WebhookEndpoint{}, nil, err
	}
	secret, err := newSigningSecret()
	if err != nil {
		return sqlc.WebhookEndpoint{}, nil, apperror.NewInternal("generate webhook secret", err)
	}
	v, e := s.repo.RotateSecret(c, org, id, secret)
	return v, secret, readErr(e, "webhook not found")
}
func (s *Service) ListDeliveries(c context.Context, org, id uuid.UUID, limit, offset int32) ([]sqlc.WebhookDelivery, error) {
	if err := validIDs(org, id); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, apperror.NewBadRequest("limit must be between 1 and 100")
	}
	if offset < 0 {
		return nil, apperror.NewBadRequest("offset must not be negative")
	}
	return s.repo.ListDeliveries(c, org, id, limit, offset)
}
func (s *Service) GetDelivery(c context.Context, org, endpoint, id uuid.UUID) (sqlc.WebhookDelivery, error) {
	if err := validIDs(org, endpoint); err != nil {
		return sqlc.WebhookDelivery{}, err
	}
	v, e := s.repo.GetDelivery(c, org, id)
	if e = readErr(e, "webhook delivery not found"); e != nil {
		return sqlc.WebhookDelivery{}, e
	}
	if v.EndpointID != endpoint {
		return sqlc.WebhookDelivery{}, apperror.NewNotFound("webhook delivery not found")
	}
	return v, nil
}
func (s *Service) Retry(c context.Context, org, endpoint, id uuid.UUID) (sqlc.WebhookDelivery, error) {
	if _, err := s.GetDelivery(c, org, endpoint, id); err != nil {
		return sqlc.WebhookDelivery{}, err
	}
	v, e := s.repo.Retry(c, org, id)
	return v, readErr(e, "webhook delivery is not retryable")
}

func (s *Service) Ingest(c context.Context, event InboundEvent) error {
	if err := validateInboundEvent(event); err != nil {
		return err
	}
	if s.db == nil {
		return fmt.Errorf("webhook database is required for event ingestion")
	}

	tx, err := s.db.Begin(c)
	if err != nil {
		return fmt.Errorf("begin webhook event transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(c) }()

	repo := s.repo.WithTx(tx)
	if _, err := repo.CreateEvent(c, event); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("create webhook event: %w", err)
		}
		existing, getErr := repo.GetEvent(c, event.OrganizationID, event.ID)
		if getErr != nil {
			return fmt.Errorf("get existing webhook event: %w", getErr)
		}
		if existing.EventType != event.EventType || existing.ObjectType != event.ObjectType {
			return fmt.Errorf("webhook event %s conflicts with existing event metadata", event.ID)
		}
	}

	if _, err := repo.CreateDeliveriesForEvent(c, event.ID, time.Now().UTC()); err != nil {
		return fmt.Errorf("create webhook deliveries: %w", err)
	}
	if err := tx.Commit(c); err != nil {
		return fmt.Errorf("commit webhook event transaction: %w", err)
	}
	return nil
}

func readErr(e error, msg string) error {
	if e == nil {
		return nil
	}
	if errors.Is(e, pgx.ErrNoRows) {
		return apperror.NewNotFound(msg)
	}
	return apperror.NewInternal(msg, e)
}
func writeErr(e error, msg string) error { return readErr(e, msg) }
func (s *Service) Test(c context.Context, org, id uuid.UUID) (int, error) {
	v, e := s.Get(c, org, id)
	if e != nil {
		return 0, e
	}
	if !v.Enabled {
		return 0, apperror.NewBadRequest("webhook is disabled")
	}
	status, e := sendTest(c, v)
	if e != nil {
		return 0, apperror.NewInternal("send webhook test", e)
	}
	return status, nil
}
