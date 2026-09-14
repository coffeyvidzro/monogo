package conferences

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
		panic("conferences: database is required")
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

func (r *Repository) Create(
	ctx context.Context,
	org uuid.UUID,
	req CreateRequest,
) (sqlc.Conference, error) {
	return r.mutateConference(
		ctx,
		EventConferenceCreated,
		func(repo *Repository) (sqlc.Conference, error) {
			return repo.queries.CreateConference(ctx, sqlc.CreateConferenceParams{
				OrganizationID: org,
				ApplicationID:  req.ApplicationID,
				Name:           req.Name,
				StartedAt: pgtype.Timestamptz{
					Time:  time.Now().UTC(),
					Valid: true,
				},
			})
		},
	)
}

func (r *Repository) Get(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.Conference, error) {
	return r.queries.GetConference(ctx, sqlc.GetConferenceParams{
		OrganizationID: org,
		ID:             id,
	})
}

func (r *Repository) List(
	ctx context.Context,
	org uuid.UUID,
	offset int32,
	limit int32,
) ([]sqlc.Conference, error) {
	return r.queries.ListConferences(ctx, sqlc.ListConferencesParams{
		OrganizationID: org,
		PageOffset:     offset,
		PageLimit:      limit,
	})
}

func (r *Repository) End(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.Conference, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.Conference{}, fmt.Errorf("begin conference end transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := r.WithTx(tx)
	conference, err := repo.queries.EndConference(ctx, sqlc.EndConferenceParams{
		OrganizationID: org,
		ID:             id,
	})
	if err != nil {
		return sqlc.Conference{}, err
	}

	participants, err := repo.queries.EndConferenceParticipants(
		ctx,
		sqlc.EndConferenceParticipantsParams{
			OrganizationID: org,
			ConferenceID:   id,
		},
	)
	if err != nil {
		return sqlc.Conference{}, err
	}

	occurredAt := time.Now().UTC()
	if err := repo.insertConferenceEvent(
		ctx,
		EventConferenceEnded,
		conference,
		occurredAt,
	); err != nil {
		return sqlc.Conference{}, err
	}

	for _, participant := range participants {
		if err := repo.insertParticipantEvent(
			ctx,
			EventParticipantLeft,
			participant,
			occurredAt,
		); err != nil {
			return sqlc.Conference{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Conference{}, fmt.Errorf("commit conference end transaction: %w", err)
	}

	return conference, nil
}

func (r *Repository) CreateParticipant(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
	req AddParticipantRequest,
) (sqlc.ConferenceParticipant, error) {
	return r.mutateParticipant(
		ctx,
		EventParticipantJoined,
		func(repo *Repository) (sqlc.ConferenceParticipant, error) {
			return repo.queries.CreateConferenceParticipant(
				ctx,
				sqlc.CreateConferenceParticipantParams{
					OrganizationID:    org,
					ConferenceID:      conferenceID,
					CallParticipantID: req.CallParticipantID,
					State:             string(ParticipantJoined),
				},
			)
		},
	)
}

func (r *Repository) GetParticipant(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.ConferenceParticipant, error) {
	return r.queries.GetConferenceParticipant(ctx, sqlc.GetConferenceParticipantParams{
		OrganizationID: org,
		ID:             id,
	})
}

func (r *Repository) ListParticipants(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
) ([]sqlc.ConferenceParticipant, error) {
	return r.queries.ListConferenceParticipants(ctx, sqlc.ListConferenceParticipantsParams{
		OrganizationID: org,
		ConferenceID:   conferenceID,
	})
}

func (r *Repository) LeaveParticipant(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.ConferenceParticipant, error) {
	return r.mutateParticipant(
		ctx,
		EventParticipantLeft,
		func(repo *Repository) (sqlc.ConferenceParticipant, error) {
			return repo.queries.LeaveConferenceParticipant(
				ctx,
				sqlc.LeaveConferenceParticipantParams{
					OrganizationID: org,
					ID:             id,
				},
			)
		},
	)
}

func (r *Repository) FailParticipant(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.ConferenceParticipant, error) {
	return r.mutateParticipant(
		ctx,
		EventParticipantFailed,
		func(repo *Repository) (sqlc.ConferenceParticipant, error) {
			return repo.queries.FailConferenceParticipant(
				ctx,
				sqlc.FailConferenceParticipantParams{
					OrganizationID: org,
					ID:             id,
				},
			)
		},
	)
}

func (r *Repository) SetParticipantMuted(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
	muted bool,
) (sqlc.ConferenceParticipant, error) {
	return r.queries.SetConferenceParticipantMuted(ctx, sqlc.SetConferenceParticipantMutedParams{
		OrganizationID: org,
		ID:             id,
		Muted:          muted,
	})
}

func (r *Repository) SetParticipantDeaf(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
	deaf bool,
) (sqlc.ConferenceParticipant, error) {
	return r.queries.SetConferenceParticipantDeaf(ctx, sqlc.SetConferenceParticipantDeafParams{
		OrganizationID: org,
		ID:             id,
		Deaf:           deaf,
	})
}

type conferenceMutation func(*Repository) (sqlc.Conference, error)

type participantMutation func(*Repository) (sqlc.ConferenceParticipant, error)

func (r *Repository) mutateConference(
	ctx context.Context,
	eventType EventType,
	mutation conferenceMutation,
) (sqlc.Conference, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.Conference{}, fmt.Errorf("begin conference transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := r.WithTx(tx)
	conference, err := mutation(repo)
	if err != nil {
		return sqlc.Conference{}, err
	}

	if err := repo.insertConferenceEvent(
		ctx,
		eventType,
		conference,
		time.Now().UTC(),
	); err != nil {
		return sqlc.Conference{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Conference{}, fmt.Errorf("commit conference transaction: %w", err)
	}

	return conference, nil
}

func (r *Repository) mutateParticipant(
	ctx context.Context,
	eventType EventType,
	mutation participantMutation,
) (sqlc.ConferenceParticipant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.ConferenceParticipant{}, fmt.Errorf(
			"begin conference participant transaction: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := r.WithTx(tx)
	participant, err := mutation(repo)
	if err != nil {
		return sqlc.ConferenceParticipant{}, err
	}

	if err := repo.insertParticipantEvent(
		ctx,
		eventType,
		participant,
		time.Now().UTC(),
	); err != nil {
		return sqlc.ConferenceParticipant{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.ConferenceParticipant{}, fmt.Errorf(
			"commit conference participant transaction: %w",
			err,
		)
	}

	return participant, nil
}

func (r *Repository) insertConferenceEvent(
	ctx context.Context,
	eventType EventType,
	conference sqlc.Conference,
	occurredAt time.Time,
) error {
	_, err := r.outbox.Insert(ctx, outbox.Event{
		Subject:       string(eventType),
		AggregateType: "conference",
		AggregateID:   conference.ID,
		Payload: Event{
			EventType:      eventType,
			OrganizationID: conference.OrganizationID,
			ConferenceID:   conference.ID,
			Resource:       conferenceResponse(conference),
			OccurredAt:     occurredAt,
		},
		Headers: eventHeaders(eventType, conference.OrganizationID),
	})
	if err != nil {
		return fmt.Errorf("insert conference outbox event: %w", err)
	}

	return nil
}

func (r *Repository) insertParticipantEvent(
	ctx context.Context,
	eventType EventType,
	participant sqlc.ConferenceParticipant,
	occurredAt time.Time,
) error {
	participantID := participant.ID
	_, err := r.outbox.Insert(ctx, outbox.Event{
		Subject:       string(eventType),
		AggregateType: "conference_participant",
		AggregateID:   participant.ID,
		Payload: Event{
			EventType:      eventType,
			OrganizationID: participant.OrganizationID,
			ConferenceID:   participant.ConferenceID,
			ParticipantID:  &participantID,
			Resource:       participantResponse(participant),
			OccurredAt:     occurredAt,
		},
		Headers: eventHeaders(eventType, participant.OrganizationID),
	})
	if err != nil {
		return fmt.Errorf("insert conference participant outbox event: %w", err)
	}

	return nil
}

func eventHeaders(eventType EventType, organizationID uuid.UUID) map[string]string {
	return map[string]string{
		"event_type":      string(eventType),
		"organization_id": organizationID.String(),
		"schema_version":  "1",
	}
}
