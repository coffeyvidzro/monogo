package recordings

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	s3integration "github.com/coffeyvidzro/monogo/internal/integrations/s3"
	platformstorage "github.com/coffeyvidzro/monogo/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ObjectStore interface {
	Put(context.Context, string, string, io.Reader, int64) error
	PlaybackURL(context.Context, string) (string, time.Time, error)
	Delete(context.Context, string) error
	Bucket() string
}

type ObjectStorage struct {
	client       ObjectStore
	integrations *platformstorage.Service
}

func NewObjectStorage(client ObjectStore) *ObjectStorage {
	if client == nil {
		panic("recordings: object store is required")
	}
	return &ObjectStorage{client: client}
}

func NewResolvedObjectStorage(
	client ObjectStore,
	integrations *platformstorage.Service,
) *ObjectStorage {
	storage := NewObjectStorage(client)
	storage.integrations = integrations
	return storage
}

func (s *ObjectStorage) ResolveUpload(
	ctx context.Context,
	recording sqlc.Recording,
) (*uuid.UUID, string, string, error) {
	client, integrationID, err := s.clientForRecording(ctx, recording, true)
	if err != nil {
		return nil, "", "", err
	}
	return integrationID, "s3", client.Bucket(), nil
}

func (s *ObjectStorage) Put(
	ctx context.Context,
	recording sqlc.Recording,
	key string,
	contentType string,
	reader io.Reader,
	size int64,
) (*uuid.UUID, string, string, error) {
	client, integrationID, err := s.clientForRecording(ctx, recording, true)
	if err != nil {
		return nil, "", "", err
	}
	if integrationID == nil {
		if err := validateManagedRecordingKey(recording.OrganizationID, key); err != nil {
			return nil, "", "", err
		}
	}
	if err := client.Put(ctx, key, contentType, reader, size); err != nil {
		return nil, "", "", err
	}
	return integrationID, "s3", client.Bucket(), nil
}

func (s *ObjectStorage) PlaybackURL(
	ctx context.Context,
	recording sqlc.Recording,
) (string, time.Time, error) {
	if recording.StorageKey == nil ||
		recording.StorageProvider == nil ||
		*recording.StorageProvider != "s3" {
		return "", time.Time{}, fmt.Errorf("recording has no S3 object")
	}
	client, integrationID, err := s.clientForRecording(ctx, recording, false)
	if err != nil {
		return "", time.Time{}, err
	}
	if recording.StorageBucket == nil || *recording.StorageBucket != client.Bucket() {
		return "", time.Time{}, fmt.Errorf("recording S3 bucket is invalid")
	}
	if integrationID == nil {
		if err := validateManagedRecordingKey(recording.OrganizationID, *recording.StorageKey); err != nil {
			return "", time.Time{}, err
		}
	}
	return client.PlaybackURL(ctx, *recording.StorageKey)
}

func (s *ObjectStorage) Delete(
	ctx context.Context,
	recording sqlc.Recording,
) error {
	if recording.StorageKey == nil ||
		recording.StorageProvider == nil ||
		*recording.StorageProvider != "s3" {
		return fmt.Errorf("recording has no S3 object")
	}
	client, integrationID, err := s.clientForRecording(ctx, recording, false)
	if err != nil {
		return err
	}
	if recording.StorageBucket == nil || *recording.StorageBucket != client.Bucket() {
		return fmt.Errorf("recording S3 bucket is invalid")
	}
	if integrationID == nil {
		if err := validateManagedRecordingKey(recording.OrganizationID, *recording.StorageKey); err != nil {
			return err
		}
	}
	return client.Delete(ctx, *recording.StorageKey)
}

func (s *ObjectStorage) clientForRecording(
	ctx context.Context,
	recording sqlc.Recording,
	forUpload bool,
) (ObjectStore, *uuid.UUID, error) {
	if recording.StorageIntegrationID != nil {
		if s.integrations == nil {
			return nil, nil, fmt.Errorf(
				"recording storage integration resolver is unavailable",
			)
		}
		resolved, err := s.integrations.ResolveByID(
			ctx,
			recording.OrganizationID,
			*recording.StorageIntegrationID,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"resolve recording storage integration: %w",
				err,
			)
		}
		client, err := customClient(ctx, resolved)
		if err != nil {
			return nil, nil, err
		}
		id := resolved.ID
		return client, &id, nil
	}

	// A storage key without an integration id means this upload was already
	// pinned to Leamout-managed storage. Do not re-resolve the organization's
	// active integration on retry.
	if forUpload && recording.StorageKey == nil && s.integrations != nil {
		resolved, err := s.integrations.ResolveRecording(
			ctx,
			recording.OrganizationID,
		)
		if err == nil {
			client, clientErr := customClient(ctx, resolved)
			if clientErr != nil {
				return nil, nil, clientErr
			}
			id := resolved.ID
			return client, &id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf(
				"resolve organization recording storage: %w",
				err,
			)
		}
	}

	return s.client, nil, nil
}

func customClient(
	ctx context.Context,
	resolved platformstorage.ResolvedIntegration,
) (ObjectStore, error) {
	client, err := s3integration.New(ctx, s3integration.Config{
		Endpoint:     resolved.EndpointURL,
		Region:       resolved.Region,
		Bucket:       resolved.Bucket,
		AccessKey:    resolved.AccessKeyID,
		SecretKey:    resolved.SecretAccessKey,
		UsePathStyle: resolved.UsePathStyle,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize recording S3 integration: %w", err)
	}
	return client, nil
}
