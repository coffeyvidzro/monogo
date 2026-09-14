package webhooks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(q *sqlc.Queries) *Repository    { return &Repository{q} }
func (r *Repository) WithTx(tx pgx.Tx) *Repository { return NewRepository(r.queries.WithTx(tx)) }
func (r *Repository) Create(c context.Context, org uuid.UUID, req CreateRequest, secret []byte) (sqlc.WebhookEndpoint, error) {
	return r.queries.CreateWebhookEndpoint(c, sqlc.CreateWebhookEndpointParams{OrganizationID: org, Url: req.URL, SigningSecret: secret, SubscribedEvents: req.SubscribedEvents, Enabled: req.Enabled})
}
func (r *Repository) List(c context.Context, org uuid.UUID) ([]sqlc.WebhookEndpoint, error) {
	return r.queries.ListWebhookEndpointsByOrganizationID(c, org)
}
func (r *Repository) Get(c context.Context, org, id uuid.UUID) (sqlc.WebhookEndpoint, error) {
	return r.queries.GetWebhookEndpointByID(c, sqlc.GetWebhookEndpointByIDParams{ID: id, OrganizationID: org})
}
func (r *Repository) Update(c context.Context, org, id uuid.UUID, req UpdateRequest) (sqlc.WebhookEndpoint, error) {
	return r.queries.UpdateWebhookEndpoint(c, sqlc.UpdateWebhookEndpointParams{ID: id, OrganizationID: org, Url: req.URL, SubscribedEvents: optionalSubscribedEvents(req.SubscribedEvents), Enabled: req.Enabled})
}
func (r *Repository) Disable(c context.Context, org, id uuid.UUID) error {
	return r.queries.DisableWebhookEndpoint(c, sqlc.DisableWebhookEndpointParams{ID: id, OrganizationID: org})
}
func (r *Repository) RotateSecret(c context.Context, org, id uuid.UUID, secret []byte) (sqlc.WebhookEndpoint, error) {
	return r.queries.RotateWebhookEndpointSecret(c, sqlc.RotateWebhookEndpointSecretParams{ID: id, OrganizationID: org, SigningSecret: secret})
}
func (r *Repository) ListDeliveries(c context.Context, org, id uuid.UUID, limit, offset int32) ([]sqlc.WebhookDelivery, error) {
	return r.queries.ListWebhookDeliveriesForEndpoint(c, sqlc.ListWebhookDeliveriesForEndpointParams{OrganizationID: org, EndpointID: id, LimitCount: limit, OffsetCount: offset})
}
func (r *Repository) GetDelivery(c context.Context, org, id uuid.UUID) (sqlc.WebhookDelivery, error) {
	return r.queries.GetWebhookDelivery(c, sqlc.GetWebhookDeliveryParams{ID: id, OrganizationID: org})
}
func (r *Repository) Retry(c context.Context, org, id uuid.UUID) (sqlc.WebhookDelivery, error) {
	return r.queries.RetryWebhookDelivery(c, sqlc.RetryWebhookDeliveryParams{ID: id, OrganizationID: org})
}
func (r *Repository) Cancel(c context.Context, org, id uuid.UUID) error {
	_, err := r.queries.CancelWebhookDeliveriesForEndpoint(c, sqlc.CancelWebhookDeliveriesForEndpointParams{EndpointID: id, OrganizationID: org})
	return err
}
func (r *Repository) CreateTestDelivery(c context.Context, org, id uuid.UUID, eventID uuid.UUID, at time.Time) (sqlc.WebhookDelivery, error) {
	return r.queries.CreateWebhookDelivery(c, sqlc.CreateWebhookDeliveryParams{EndpointID: id, EventID: eventID, NextAttemptAt: pgtype.Timestamptz{Time: at, Valid: true}})
}

func (r *Repository) CreateEvent(c context.Context, event InboundEvent) (sqlc.WebhookEvent, error) {
	return r.queries.CreateWebhookEvent(c, sqlc.CreateWebhookEventParams{
		ID:             event.ID,
		OrganizationID: event.OrganizationID,
		EventType:      event.EventType,
		ObjectType:     event.ObjectType,
		ObjectID:       event.ObjectID,
		Payload:        event.Payload,
		OccurredAt:     pgtype.Timestamptz{Time: event.OccurredAt, Valid: true},
	})
}

func (r *Repository) GetEvent(c context.Context, org, id uuid.UUID) (sqlc.WebhookEvent, error) {
	return r.queries.GetWebhookEvent(c, sqlc.GetWebhookEventParams{ID: id, OrganizationID: org})
}

func (r *Repository) CreateDeliveriesForEvent(c context.Context, id uuid.UUID, at time.Time) (int64, error) {
	return r.queries.CreateWebhookDeliveriesForEvent(c, sqlc.CreateWebhookDeliveriesForEventParams{
		EventID:       id,
		NextAttemptAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

func (r *Repository) ClaimDeliveries(c context.Context, workerID string, staleBefore time.Time, limit int32) ([]sqlc.ClaimWebhookDeliveriesRow, error) {
	return r.queries.ClaimWebhookDeliveries(c, sqlc.ClaimWebhookDeliveriesParams{
		WorkerID:    &workerID,
		StaleBefore: pgtype.Timestamptz{Time: staleBefore, Valid: true},
		LimitCount:  limit,
	})
}

func (r *Repository) MarkDeliverySucceeded(c context.Context, workerID string, id uuid.UUID, attempt DeliveryAttempt) error {
	_, err := r.queries.MarkWebhookDeliverySucceeded(c, sqlc.MarkWebhookDeliverySucceededParams{
		ResponseStatus: attempt.StatusCode,
		ResponseBody:   attempt.Body,
		ID:             id,
		WorkerID:       &workerID,
	})
	return err
}

func (r *Repository) ScheduleDeliveryRetry(c context.Context, workerID string, id uuid.UUID, at time.Time, attempt DeliveryAttempt) error {
	message := attemptError(attempt)
	_, err := r.queries.ScheduleWebhookDeliveryRetry(c, sqlc.ScheduleWebhookDeliveryRetryParams{
		NextAttemptAt:  pgtype.Timestamptz{Time: at, Valid: true},
		ResponseStatus: attempt.StatusCode,
		ResponseBody:   attempt.Body,
		LastError:      &message,
		ID:             id,
		WorkerID:       &workerID,
	})
	return err
}

func (r *Repository) MarkDeliveryFailed(c context.Context, workerID string, id uuid.UUID, attempt DeliveryAttempt, autoDisableAfter int32) error {
	message := attemptError(attempt)
	_, err := r.queries.MarkWebhookDeliveryFailed(c, sqlc.MarkWebhookDeliveryFailedParams{
		ResponseStatus:   attempt.StatusCode,
		ResponseBody:     attempt.Body,
		LastError:        &message,
		ID:               id,
		WorkerID:         &workerID,
		AutoDisableAfter: autoDisableAfter,
	})
	return err
}

func optionalSubscribedEvents(events *[]string) []string {
	if events == nil {
		return nil
	}
	return *events
}

func attemptError(attempt DeliveryAttempt) string {
	if attempt.Err != nil {
		return attempt.Err.Error()
	}
	if attempt.StatusCode != nil {
		return fmt.Sprintf("webhook endpoint returned HTTP %d", *attempt.StatusCode)
	}
	return "webhook delivery failed"
}
