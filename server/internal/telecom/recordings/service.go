package recordings

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

// Storage manages recording objects without exposing storage URLs in metadata.
type Storage interface {
	PlaybackURL(context.Context, sqlc.Recording) (string, time.Time, error)
	Delete(context.Context, sqlc.Recording) error
}

type Service struct {
	repo    *Repository
	storage Storage
}

func NewService(repo *Repository, storage Storage) *Service {
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
	channelID := strings.TrimSpace(event.ChannelID)
	path := strings.TrimSpace(event.Path)
	if channelID == "" || path == "" {
		return apperror.NewBadRequest("recording lifecycle event requires channel id and path")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	call, err := s.repo.GetCallByChannelID(ctx, channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("resolve recording call", err)
	}

	_, err = s.repo.GetByCallStorageKey(ctx, call.ID, path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewInternal("get recording by media path", err)
	}

	_, err = s.repo.Start(ctx, call, path, event.OccurredAt.UTC())
	if err != nil {
		// A duplicated FreeSWITCH event may race another worker event. Re-read the
		// stable (call_id, storage_key) identity before surfacing an error.
		if _, readErr := s.repo.GetByCallStorageKey(ctx, call.ID, path); readErr == nil {
			return nil
		}
		return apperror.NewInternal("start recording lifecycle", err)
	}
	return nil
}

func (s *Service) ObserveStopped(ctx context.Context, event LifecycleEvent) error {
	channelID := strings.TrimSpace(event.ChannelID)
	path := strings.TrimSpace(event.Path)
	if channelID == "" || path == "" {
		return apperror.NewBadRequest("recording lifecycle event requires channel id and path")
	}

	call, err := s.repo.GetCallByChannelID(ctx, channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("resolve recording call", err)
	}

	recording, err := s.repo.GetByCallStorageKey(ctx, call.ID, path)
	if errors.Is(err, pgx.ErrNoRows) {
		if startErr := s.ObserveStarted(ctx, event); startErr != nil {
			return startErr
		}
		recording, err = s.repo.GetByCallStorageKey(ctx, call.ID, path)
	}
	if err != nil {
		return apperror.NewInternal("get recording by media path", err)
	}
	if recording.Status == string(StatusCompleted) || recording.Status == string(StatusFailed) || recording.Status == string(StatusDeleted) {
		return nil
	}

	_, err = s.repo.Complete(ctx, recording)
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
	if recording.Status == string(StatusRecording) {
		return apperror.NewConflict("recording cannot be deleted while recording")
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
