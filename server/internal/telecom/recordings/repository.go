package recordings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/modules/outbox"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
	outbox  *outbox.Repository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	if db == nil {
		panic("recordings: database is required")
	}

	queries := sqlc.New(db)
	return &Repository{
		db:      db,
		queries: queries,
		outbox:  outbox.NewRepository(queries),
	}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	queries := r.queries.WithTx(tx)
	return &Repository{
		db:      r.db,
		queries: queries,
		outbox:  outbox.NewRepository(queries),
	}
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Recording, error) {
	return r.queries.GetRecording(ctx, sqlc.GetRecordingParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) GetIncludingDeleted(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Recording, error) {
	return r.queries.GetRecordingIncludingDeleted(ctx, sqlc.GetRecordingIncludingDeletedParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) GetByCallStorageKey(
	ctx context.Context,
	callID uuid.UUID,
	storageKey string,
) (sqlc.Recording, error) {
	return r.queries.GetRecordingByCallStorageKey(ctx, sqlc.GetRecordingByCallStorageKeyParams{
		CallID:     callID,
		StorageKey: &storageKey,
	})
}

func (r *Repository) GetCallByChannelID(
	ctx context.Context,
	channelID string,
) (sqlc.Call, error) {
	return r.queries.GetCallBySIPCallIDGlobal(ctx, &channelID)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	offset int32,
	limit int32,
) ([]sqlc.Recording, error) {
	return r.queries.ListRecordings(ctx, sqlc.ListRecordingsParams{
		OrganizationID: organizationID,
		PageOffset:     offset,
		PageLimit:      limit,
	})
}

func (r *Repository) ListForReconciliation(
	ctx context.Context,
	updatedBefore time.Time,
	batchSize int32,
) ([]sqlc.Recording, error) {
	return r.queries.ListRecordingsForReconciliation(ctx, sqlc.ListRecordingsForReconciliationParams{
		UpdatedBefore: pgtype.Timestamptz{
			Time:  updatedBefore,
			Valid: true,
		},
		BatchSize: batchSize,
	})
}

func (r *Repository) Start(
	ctx context.Context,
	call sqlc.Call,
	path string,
	occurredAt time.Time,
) (sqlc.Recording, error) {
	return r.mutate(
		ctx,
		EventRecordingStarted,
		func(repo *Repository) (sqlc.Recording, error) {
			provider := "freeswitch-local"
			return repo.queries.CreateRecording(ctx, sqlc.CreateRecordingParams{
				OrganizationID:  call.OrganizationID,
				CallID:          call.ID,
				Status:          string(StatusRecording),
				StorageKey:      &path,
				StorageProvider: &provider,
				StartedAt: pgtype.Timestamptz{
					Time:  occurredAt,
					Valid: true,
				},
			})
		},
	)
}

func (r *Repository) Complete(
	ctx context.Context,
	recording sqlc.Recording,
) (sqlc.Recording, error) {
	return r.mutate(
		ctx,
		EventRecordingCompleted,
		func(repo *Repository) (sqlc.Recording, error) {
			return repo.queries.CompleteRecording(ctx, sqlc.CompleteRecordingParams{
				OrganizationID: recording.OrganizationID,
				ID:             recording.ID,
			})
		},
	)
}

func (r *Repository) Fail(
	ctx context.Context,
	recording sqlc.Recording,
) (sqlc.Recording, error) {
	return r.mutate(
		ctx,
		EventRecordingFailed,
		func(repo *Repository) (sqlc.Recording, error) {
			return repo.queries.FailRecording(ctx, sqlc.FailRecordingParams{
				OrganizationID: recording.OrganizationID,
				ID:             recording.ID,
			})
		},
	)
}

func (r *Repository) Delete(
	ctx context.Context,
	recording sqlc.Recording,
) (sqlc.Recording, error) {
	return r.mutate(
		ctx,
		EventRecordingDeleted,
		func(repo *Repository) (sqlc.Recording, error) {
			return repo.queries.DeleteRecording(ctx, sqlc.DeleteRecordingParams{
				OrganizationID: recording.OrganizationID,
				ID:             recording.ID,
			})
		},
	)
}

type recordingMutation func(*Repository) (sqlc.Recording, error)

func (r *Repository) mutate(
	ctx context.Context,
	eventType EventType,
	mutation recordingMutation,
) (sqlc.Recording, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.Recording{}, fmt.Errorf("begin recording transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := r.WithTx(tx)
	recording, err := mutation(repo)
	if err != nil {
		return sqlc.Recording{}, err
	}

	occurredAt := time.Now().UTC()
	if _, err := repo.outbox.Insert(ctx, outbox.Event{
		Subject:       string(eventType),
		AggregateType: "recording",
		AggregateID:   recording.ID,
		Payload: Event{
			EventType:      eventType,
			OrganizationID: recording.OrganizationID,
			RecordingID:    recording.ID,
			CallID:         recording.CallID,
			Resource:       recordingResponse(recording),
			OccurredAt:     occurredAt,
		},
		Headers: map[string]string{
			"event_type":      string(eventType),
			"organization_id": recording.OrganizationID.String(),
			"schema_version":  "1",
		},
	}); err != nil {
		return sqlc.Recording{}, fmt.Errorf("insert recording outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Recording{}, fmt.Errorf("commit recording transaction: %w", err)
	}

	return recording, nil
}
