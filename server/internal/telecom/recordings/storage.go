package recordings

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
)

type ObjectStore interface {
	Put(context.Context, string, string, io.Reader, int64) error
	PlaybackURL(context.Context, string) (string, time.Time, error)
	Delete(context.Context, string) error
	Bucket() string
}

type ObjectStorage struct{ client ObjectStore }

func NewObjectStorage(client ObjectStore) *ObjectStorage {
	if client == nil {
		panic("recordings: object store is required")
	}
	return &ObjectStorage{client: client}
}

func (s *ObjectStorage) PlaybackURL(ctx context.Context, recording sqlc.Recording) (string, time.Time, error) {
	if recording.StorageKey == nil || recording.StorageProvider == nil || *recording.StorageProvider != "s3" {
		return "", time.Time{}, fmt.Errorf("recording has no S3 object")
	}
	if recording.StorageBucket == nil || *recording.StorageBucket != s.client.Bucket() {
		return "", time.Time{}, fmt.Errorf("recording S3 bucket is invalid")
	}
	return s.client.PlaybackURL(ctx, *recording.StorageKey)
}

func (s *ObjectStorage) Delete(ctx context.Context, recording sqlc.Recording) error {
	if recording.StorageKey == nil || recording.StorageProvider == nil || *recording.StorageProvider != "s3" {
		return fmt.Errorf("recording has no S3 object")
	}
	if recording.StorageBucket == nil || *recording.StorageBucket != s.client.Bucket() {
		return fmt.Errorf("recording S3 bucket is invalid")
	}
	return s.client.Delete(ctx, *recording.StorageKey)
}
