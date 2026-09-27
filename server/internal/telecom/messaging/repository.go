package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
	outbox  *outbox.Repository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	if db == nil {
		panic("messaging: database is required")
	}
	queries := sqlc.New(db)
	return &Repository{db: db, queries: queries, outbox: outbox.NewRepository(queries)}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	queries := r.queries.WithTx(tx)
	return &Repository{db: r.db, queries: queries, outbox: outbox.NewRepository(queries)}
}

func (r *Repository) CreateOutbound(
	ctx context.Context,
	organizationID uuid.UUID,
	idempotencyKey, requestHash string,
	req CreateRequest,
) (sqlc.Message, error) {
	media, err := marshalMedia(req.Media)
	if err != nil {
		return sqlc.Message{}, err
	}
	return r.mutate(ctx, EventQueued, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.CreateOutboundMessage(ctx, sqlc.CreateOutboundMessageParams{
			OrganizationID: organizationID,
			Channel:        string(req.Channel),
			FromAddress:    req.From,
			ToAddress:      req.To,
			Body:           req.Body,
			Media:          media,
			IdempotencyKey: &idempotencyKey,
			RequestHash:    &requestHash,
		})
	})
}

func (r *Repository) CreateInbound(ctx context.Context, req InboundRequest) (sqlc.Message, error) {
	media, err := marshalMedia(req.Media)
	if err != nil {
		return sqlc.Message{}, err
	}
	return r.mutate(ctx, EventReceived, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.CreateInboundMessage(ctx, sqlc.CreateInboundMessageParams{
			OrganizationID:        req.OrganizationID,
			MessagingConnectionID: req.MessagingConnectionID,
			Channel:               string(req.Channel),
			FromAddress:           req.From,
			ToAddress:             req.To,
			Body:                  req.Body,
			Media:                 media,
			ProviderMessageID:     &req.ProviderMessageID,
		})
	})
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Message, error) {
	return r.queries.GetMessage(ctx, sqlc.GetMessageParams{OrganizationID: organizationID, ID: id})
}

func (r *Repository) GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, key string) (sqlc.Message, error) {
	return r.queries.GetMessageByIdempotencyKey(ctx, sqlc.GetMessageByIdempotencyKeyParams{
		OrganizationID: organizationID,
		IdempotencyKey: &key,
	})
}

func (r *Repository) GetByProviderID(ctx context.Context, messagingConnectionID uuid.UUID, providerMessageID string) (sqlc.Message, error) {
	return r.queries.GetMessageByProviderID(ctx, sqlc.GetMessageByProviderIDParams{
		MessagingConnectionID: &messagingConnectionID,
		ProviderMessageID:     &providerMessageID,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID, req ListRequest) ([]sqlc.Message, error) {
	return r.queries.ListMessages(ctx, sqlc.ListMessagesParams{
		OrganizationID: organizationID,
		Status:         req.Status,
		Direction:      req.Direction,
		Channel:        req.Channel,
		PageOffset:     req.Offset,
		PageLimit:      req.Limit,
	})
}

func (r *Repository) ResolveConnection(ctx context.Context, organizationID uuid.UUID, channel Channel) (sqlc.MessagingConnection, error) {
	return r.queries.ResolveMessagingConnection(ctx, sqlc.ResolveMessagingConnectionParams{
		OrganizationID: organizationID,
		Channel:        string(channel),
	})
}

func (r *Repository) SetProviderAttribution(ctx context.Context, organizationID, id, messagingConnectionID uuid.UUID) (sqlc.Message, error) {
	return r.queries.SetMessageProviderAttribution(ctx, sqlc.SetMessageProviderAttributionParams{
		MessagingConnectionID: messagingConnectionID,
		OrganizationID:        organizationID,
		ID:                    id,
	})
}

func (r *Repository) MarkSubmitted(ctx context.Context, organizationID, id uuid.UUID, providerMessageID string) (sqlc.Message, error) {
	return r.mutate(ctx, EventSubmitted, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.MarkMessageSubmitted(ctx, sqlc.MarkMessageSubmittedParams{
			ProviderMessageID: &providerMessageID,
			OrganizationID:    organizationID,
			ID:                id,
		})
	})
}

func (r *Repository) MarkSent(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Message, error) {
	return r.mutate(ctx, EventSent, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.MarkMessageSent(ctx, sqlc.MarkMessageSentParams{OrganizationID: organizationID, ID: id})
	})
}

func (r *Repository) MarkDelivered(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Message, error) {
	return r.mutate(ctx, EventDelivered, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.MarkMessageDelivered(ctx, sqlc.MarkMessageDeliveredParams{OrganizationID: organizationID, ID: id})
	})
}

func (r *Repository) MarkUndelivered(ctx context.Context, organizationID, id uuid.UUID, failure Failure) (sqlc.Message, error) {
	return r.mutate(ctx, EventUndelivered, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.MarkMessageUndelivered(ctx, sqlc.MarkMessageUndeliveredParams{
			FailureCode:    failure.Code,
			FailureMessage: failure.Message,
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkFailed(ctx context.Context, organizationID, id uuid.UUID, failure Failure) (sqlc.Message, error) {
	return r.mutate(ctx, EventFailed, func(repo *Repository) (sqlc.Message, error) {
		return repo.queries.MarkMessageFailed(ctx, sqlc.MarkMessageFailedParams{
			FailureCode:    failure.Code,
			FailureMessage: failure.Message,
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

type mutation func(*Repository) (sqlc.Message, error)

func (r *Repository) mutate(ctx context.Context, eventType EventType, fn mutation) (sqlc.Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("begin message transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := r.WithTx(tx)
	value, err := fn(repo)
	if err != nil {
		return sqlc.Message{}, err
	}
	if err := repo.insertEvent(ctx, eventType, value, time.Now().UTC()); err != nil {
		return sqlc.Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Message{}, fmt.Errorf("commit message transaction: %w", err)
	}
	return value, nil
}

func (r *Repository) insertEvent(ctx context.Context, eventType EventType, value sqlc.Message, occurredAt time.Time) error {
	_, err := r.outbox.Insert(ctx, outbox.Event{
		Subject:       string(eventType),
		AggregateType: "message",
		AggregateID:   value.ID,
		Payload: Event{
			EventType:      eventType,
			OrganizationID: value.OrganizationID,
			MessageID:      value.ID,
			Resource:       messageResponse(value),
			OccurredAt:     occurredAt,
		},
		Headers: map[string]string{
			"event_type":      string(eventType),
			"organization_id": value.OrganizationID.String(),
			"schema_version":  "1",
		},
	})
	if err != nil {
		return fmt.Errorf("insert message outbox event: %w", err)
	}
	return nil
}
