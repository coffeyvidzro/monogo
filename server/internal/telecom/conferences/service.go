package conferences

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct {
	repo       *Repository
	controller Controller
}

func NewService(repo *Repository, controller Controller) *Service {
	if repo == nil {
		panic("conferences: repository is required")
	}
	if controller == nil {
		panic("conferences: media controller is required")
	}

	return &Service{
		repo:       repo,
		controller: controller,
	}
}

func (s *Service) Create(
	ctx context.Context,
	org uuid.UUID,
	req CreateRequest,
) (sqlc.Conference, error) {
	if err := validateOrganizationID(org); err != nil {
		return sqlc.Conference{}, err
	}
	if err := validateCreateRequest(&req); err != nil {
		return sqlc.Conference{}, err
	}

	value, err := s.repo.Create(ctx, org, req)
	return value, conferenceWriteError(err, "create conference")
}

func (s *Service) Get(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.Conference, error) {
	if err := validateOrganizationID(org); err != nil {
		return sqlc.Conference{}, err
	}

	value, err := s.repo.Get(ctx, org, id)
	return value, conferenceReadError(err, "conference not found")
}

func (s *Service) List(
	ctx context.Context,
	org uuid.UUID,
	offset int32,
	limit int32,
) ([]sqlc.Conference, error) {
	if err := validateOrganizationID(org); err != nil {
		return nil, err
	}
	if offset < 0 {
		return nil, apperror.NewBadRequest("offset cannot be negative")
	}
	if limit < 1 || limit > 100 {
		return nil, apperror.NewBadRequest("limit must be between 1 and 100")
	}

	values, err := s.repo.List(ctx, org, offset, limit)
	if err != nil {
		return nil, apperror.NewInternal("list conferences", err)
	}

	return values, nil
}

func (s *Service) End(
	ctx context.Context,
	org uuid.UUID,
	id uuid.UUID,
) (sqlc.Conference, error) {
	conference, err := s.Get(ctx, org, id)
	if err != nil {
		return sqlc.Conference{}, err
	}
	if conference.State == string(StateEnded) {
		return conference, nil
	}
	if conference.State != string(StateActive) {
		return sqlc.Conference{}, apperror.NewConflict(
			"conference cannot be ended from state: " + conference.State,
		)
	}

	value, err := s.repo.End(ctx, org, id)
	if err == nil {
		return value, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		current, readErr := s.Get(ctx, org, id)
		if readErr != nil {
			return sqlc.Conference{}, readErr
		}
		if current.State == string(StateEnded) {
			return current, nil
		}
	}

	return sqlc.Conference{}, conferenceWriteError(err, "end conference")
}

func (s *Service) Lock(ctx context.Context, org uuid.UUID, id uuid.UUID) error {
	conference, err := s.Get(ctx, org, id)
	if err != nil {
		return err
	}
	if conference.State != string(StateActive) {
		return apperror.NewConflict("conference has ended")
	}
	if err := s.controller.Lock(ctx, conference.Name); err != nil {
		return apperror.NewInternal("lock conference media", err)
	}
	return nil
}

func (s *Service) Unlock(ctx context.Context, org uuid.UUID, id uuid.UUID) error {
	conference, err := s.Get(ctx, org, id)
	if err != nil {
		return err
	}
	if conference.State != string(StateActive) {
		return apperror.NewConflict("conference has ended")
	}
	if err := s.controller.Unlock(ctx, conference.Name); err != nil {
		return apperror.NewInternal("unlock conference media", err)
	}
	return nil
}

func (s *Service) AddParticipant(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
	req AddParticipantRequest,
) (sqlc.ConferenceParticipant, error) {
	conference, err := s.Get(ctx, org, conferenceID)
	if err != nil {
		return sqlc.ConferenceParticipant{}, err
	}
	if conference.State != string(StateActive) {
		return sqlc.ConferenceParticipant{}, apperror.NewConflict("conference has ended")
	}
	if err := validateParticipantRequest(&req); err != nil {
		return sqlc.ConferenceParticipant{}, err
	}

	value, err := s.repo.CreateParticipant(ctx, org, conferenceID, req)
	return value, conferenceWriteError(err, "add conference participant")
}

func (s *Service) ListParticipants(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
) ([]sqlc.ConferenceParticipant, error) {
	if _, err := s.Get(ctx, org, conferenceID); err != nil {
		return nil, err
	}

	values, err := s.repo.ListParticipants(ctx, org, conferenceID)
	if err != nil {
		return nil, apperror.NewInternal("list conference participants", err)
	}

	return values, nil
}

func (s *Service) participant(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
	id uuid.UUID,
) (sqlc.ConferenceParticipant, error) {
	if _, err := s.Get(ctx, org, conferenceID); err != nil {
		return sqlc.ConferenceParticipant{}, err
	}

	value, err := s.repo.GetParticipant(ctx, org, id)
	if err = conferenceReadError(err, "conference participant not found"); err != nil {
		return sqlc.ConferenceParticipant{}, err
	}
	if value.ConferenceID != conferenceID {
		return sqlc.ConferenceParticipant{}, apperror.NewNotFound("conference participant not found")
	}

	return value, nil
}

func (s *Service) SetParticipant(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
	id uuid.UUID,
	state string,
	muted *bool,
	deaf *bool,
) (sqlc.ConferenceParticipant, error) {
	participant, err := s.participant(ctx, org, conferenceID, id)
	if err != nil {
		return sqlc.ConferenceParticipant{}, err
	}

	if state == string(ParticipantLeft) {
		if participant.State == string(ParticipantLeft) {
			return participant, nil
		}
		if participant.State == string(ParticipantFailed) {
			return sqlc.ConferenceParticipant{}, apperror.NewConflict(
				"failed conference participant cannot leave",
			)
		}

		value, err := s.repo.LeaveParticipant(ctx, org, id)
		return s.resolveParticipantMutation(
			ctx,
			org,
			conferenceID,
			id,
			value,
			err,
			string(ParticipantLeft),
			"remove conference participant",
		)
	}

	if state == string(ParticipantFailed) {
		if participant.State == string(ParticipantFailed) {
			return participant, nil
		}
		if participant.State == string(ParticipantLeft) {
			return sqlc.ConferenceParticipant{}, apperror.NewConflict(
				"left conference participant cannot fail",
			)
		}

		value, err := s.repo.FailParticipant(ctx, org, id)
		return s.resolveParticipantMutation(
			ctx,
			org,
			conferenceID,
			id,
			value,
			err,
			string(ParticipantFailed),
			"fail conference participant",
		)
	}

	if participant.State != string(ParticipantJoined) {
		return sqlc.ConferenceParticipant{}, apperror.NewConflict(
			"conference participant is not joined",
		)
	}

	if muted != nil {
		if participant.Muted == *muted {
			return participant, nil
		}

		value, err := s.repo.SetParticipantMuted(ctx, org, id, *muted)
		return s.resolveParticipantMutation(
			ctx,
			org,
			conferenceID,
			id,
			value,
			err,
			string(ParticipantJoined),
			"update conference participant mute state",
		)
	}

	if deaf != nil {
		if participant.Deaf == *deaf {
			return participant, nil
		}

		value, err := s.repo.SetParticipantDeaf(ctx, org, id, *deaf)
		return s.resolveParticipantMutation(
			ctx,
			org,
			conferenceID,
			id,
			value,
			err,
			string(ParticipantJoined),
			"update conference participant deaf state",
		)
	}

	return participant, nil
}

func (s *Service) resolveParticipantMutation(
	ctx context.Context,
	org uuid.UUID,
	conferenceID uuid.UUID,
	id uuid.UUID,
	value sqlc.ConferenceParticipant,
	err error,
	desiredState string,
	message string,
) (sqlc.ConferenceParticipant, error) {
	if err == nil {
		return value, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		current, readErr := s.participant(ctx, org, conferenceID, id)
		if readErr != nil {
			return sqlc.ConferenceParticipant{}, readErr
		}
		if current.State == desiredState {
			return current, nil
		}
	}

	return sqlc.ConferenceParticipant{}, conferenceWriteError(err, message)
}

func conferenceReadError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	if err != nil {
		return apperror.NewInternal("get conference", err)
	}

	return nil
}

func conferenceWriteError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewConflict(
			message + " is not valid in the current lifecycle state",
		)
	}
	if err != nil {
		return apperror.NewInternal(message, err)
	}

	return nil
}
