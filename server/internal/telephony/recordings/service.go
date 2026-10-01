package recordings

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Storage manages recording objects without exposing storage URLs in metadata.
type Storage interface {
	PlaybackURL(context.Context, sqlc.Recording) (string, time.Time, error)
	Delete(context.Context, sqlc.Recording) error
}

type recordingRepository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (sqlc.Recording, error)
	GetIncludingDeleted(context.Context, uuid.UUID, uuid.UUID) (sqlc.Recording, error)
	GetByCallStorageKey(context.Context, uuid.UUID, string) (sqlc.Recording, error)
	GetCallOrganizationID(context.Context, uuid.UUID) (uuid.UUID, error)
	List(context.Context, uuid.UUID, int32, int32) ([]sqlc.Recording, error)
	Start(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (sqlc.Recording, error)
	MarkReadyForUpload(context.Context, sqlc.Recording, time.Time) (sqlc.Recording, error)
	Delete(context.Context, sqlc.Recording) (sqlc.Recording, error)
}

type Service struct {
	repo    recordingRepository
	storage Storage
}

func NewService(repo recordingRepository, storage Storage) *Service {
	if repo == nil {
		panic("recordings: repository is required")
	}
	return &Service{repo: repo, storage: storage}
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Recording, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.Recording{}, err
	}
	recording, err := s.repo.Get(ctx, organizationID, id)
	return recording, readError(err)
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID, offset, limit int32) ([]sqlc.Recording, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}
	if err := validatePagination(offset, limit); err != nil {
		return nil, err
	}
	recordings, err := s.repo.List(ctx, organizationID, offset, limit)
	if err != nil {
		return nil, apperror.NewInternal("list recordings", err)
	}
	return recordings, nil
}

func (s *Service) ObserveStarted(ctx context.Context, event LifecycleEvent) error {
	path := strings.TrimSpace(event.Path)
	if event.CallID == uuid.Nil || path == "" {
		return apperror.NewBadRequest("recording lifecycle event requires call id and path")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	organizationID, err := s.repo.GetCallOrganizationID(ctx, event.CallID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("resolve recording call", err)
	}

	_, err = s.repo.GetByCallStorageKey(ctx, event.CallID, path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewInternal("get recording by media path", err)
	}

	_, err = s.repo.Start(ctx, organizationID, event.CallID, path, event.OccurredAt.UTC())
	if err != nil {
		if _, readErr := s.repo.GetByCallStorageKey(ctx, event.CallID, path); readErr == nil {
			return nil
		}
		return apperror.NewInternal("start recording lifecycle", err)
	}
	return nil
}

func (s *Service) ObserveStopped(ctx context.Context, event LifecycleEvent) error {
	path := strings.TrimSpace(event.Path)
	if event.CallID == uuid.Nil || path == "" {
		return apperror.NewBadRequest("recording lifecycle event requires call id and path")
	}

	recording, err := s.repo.GetByCallStorageKey(ctx, event.CallID, path)
	if errors.Is(err, pgx.ErrNoRows) {
		if startErr := s.ObserveStarted(ctx, event); startErr != nil {
			return startErr
		}
		recording, err = s.repo.GetByCallStorageKey(ctx, event.CallID, path)
	}
	if err != nil {
		return apperror.NewInternal("get recording by media path", err)
	}
	if recording.Status == string(StatusUploading) || recording.Status == string(StatusCompleted) || recording.Status == string(StatusFailed) || recording.Status == string(StatusDeleted) {
		return nil
	}

	stoppedAt := event.OccurredAt
	if stoppedAt.IsZero() {
		stoppedAt = time.Now().UTC()
	}
	_, err = s.repo.MarkReadyForUpload(ctx, recording, stoppedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("complete recording lifecycle", err)
	}
	return nil
}

func (s *Service) Playback(ctx context.Context, organizationID, id uuid.UUID) (PlaybackResponse, error) {
	recording, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return PlaybackResponse{}, err
	}
	if recording.Status != string(StatusCompleted) {
		return PlaybackResponse{}, apperror.NewConflict("recording is not available for playback")
	}
	if s.storage == nil {
		return PlaybackResponse{}, apperror.NewServiceUnavailable("recording storage is unavailable", nil)
	}
	url, expiresAt, err := s.storage.PlaybackURL(ctx, recording)
	if err != nil {
		return PlaybackResponse{}, apperror.NewServiceUnavailable("create recording playback URL", err)
	}
	if url == "" || expiresAt.IsZero() {
		return PlaybackResponse{}, apperror.NewServiceUnavailable("recording storage returned an invalid playback URL", nil)
	}
	return PlaybackResponse{URL: url, ExpiresAt: expiresAt.UTC()}, nil
}

func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	if err := validateOrganizationID(organizationID); err != nil {
		return err
	}

	recording, err := s.repo.GetIncludingDeleted(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("recording not found")
	}
	if err != nil {
		return apperror.NewInternal("get recording", err)
	}
	if recording.Status == string(StatusDeleted) {
		return nil
	}
	if recording.Status == string(StatusRecording) || recording.Status == string(StatusUploading) {
		return apperror.NewConflict("recording cannot be deleted before upload completes")
	}
	if s.storage == nil {
		return apperror.NewServiceUnavailable("recording storage is unavailable", nil)
	}
	if err := s.storage.Delete(ctx, recording); err != nil {
		return apperror.NewServiceUnavailable("delete recording object", err)
	}

	_, err = s.repo.Delete(ctx, recording)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("delete recording", err)
	}
	return nil
}

func readError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("recording not found")
	}
	if err != nil {
		return apperror.NewInternal("get recording", err)
	}
	return nil
}
