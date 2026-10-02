package recordings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type fakeIngestionRepository struct {
	items                              []sqlc.Recording
	pinned, completed, retries, failed int
}

func (f *fakeIngestionRepository) ListForUpload(
	context.Context,
	time.Time,
	time.Time,
	int32,
) ([]sqlc.Recording, error) {
	return f.items, nil
}

func (f *fakeIngestionRepository) PinUpload(
	_ context.Context,
	item sqlc.Recording,
	storageIntegrationID *uuid.UUID,
	key string,
	provider string,
	bucket string,
) (sqlc.Recording, error) {
	f.pinned++
	item.StorageIntegrationID = storageIntegrationID
	item.StorageKey = &key
	item.StorageProvider = &provider
	item.StorageBucket = &bucket
	return item, nil
}

func (f *fakeIngestionRepository) CompleteUpload(
	_ context.Context,
	item sqlc.Recording,
	storageIntegrationID *uuid.UUID,
	key string,
	provider string,
	bucket string,
	format string,
	size int64,
) (sqlc.Recording, error) {
	f.completed++
	item.StorageIntegrationID = storageIntegrationID
	item.StorageKey = &key
	item.StorageProvider = &provider
	item.StorageBucket = &bucket
	item.Format = &format
	item.FileSizeBytes = &size
	item.Status = string(StatusCompleted)
	return item, nil
}

func (f *fakeIngestionRepository) RetryUpload(
	context.Context,
	sqlc.Recording,
	time.Time,
	string,
) error {
	f.retries++
	return nil
}

func (f *fakeIngestionRepository) Fail(
	_ context.Context,
	item sqlc.Recording,
) (sqlc.Recording, error) {
	f.failed++
	item.Status = string(StatusFailed)
	return item, nil
}

func TestIngestionUploadsAndRemovesStagedFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "call.wav")
	if err := os.WriteFile(path, []byte("RIFF-recording"), 0o600); err != nil {
		t.Fatal(err)
	}
	recording := sqlc.Recording{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		SourcePath:     &path,
		Status:         "uploading",
	}
	repo := &fakeIngestionRepository{items: []sqlc.Recording{recording}}
	object := &fakeObjectStore{}
	job, err := NewIngestionJob(
		repo,
		NewObjectStorage(object),
		DefaultIngestionConfig(root),
	)
	if err != nil {
		t.Fatal(err)
	}
	job.now = func() time.Time {
		return time.Date(2026, 9, 20, 1, 2, 3, 0, time.UTC)
	}
	if err := job.Ingest(context.Background()); err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if repo.pinned != 1 || repo.completed != 1 || string(object.putBody) != "RIFF-recording" {
		t.Fatalf("pinned=%d completed=%d body=%q", repo.pinned, repo.completed, object.putBody)
	}
	wantPrefix := "organizations/" + recording.OrganizationID.String() + "/recordings/2026/09/20/"
	if !strings.HasPrefix(object.putKey, wantPrefix) {
		t.Fatalf("put key = %q, want prefix %q", object.putKey, wantPrefix)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged file still exists: %v", err)
	}
}

func TestIngestionReusesPinnedStorageKey(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "call.wav")
	if err := os.WriteFile(path, []byte("RIFF-recording"), 0o600); err != nil {
		t.Fatal(err)
	}
	organizationID, recordingID := uuid.New(), uuid.New()
	key, err := recordingObjectKey(
		organizationID,
		recordingID,
		time.Date(2026, 9, 19, 1, 2, 3, 0, time.UTC),
		"wav",
	)
	if err != nil {
		t.Fatal(err)
	}
	provider, bucket := "s3", "recordings"
	recording := sqlc.Recording{
		ID:              recordingID,
		OrganizationID:  organizationID,
		SourcePath:      &path,
		Status:          "uploading",
		StorageKey:      &key,
		StorageProvider: &provider,
		StorageBucket:   &bucket,
	}
	repo := &fakeIngestionRepository{items: []sqlc.Recording{recording}}
	object := &fakeObjectStore{}
	job, err := NewIngestionJob(
		repo,
		NewObjectStorage(object),
		DefaultIngestionConfig(root),
	)
	if err != nil {
		t.Fatal(err)
	}
	job.now = func() time.Time {
		return time.Date(2026, 9, 20, 1, 2, 3, 0, time.UTC)
	}
	if err := job.Ingest(context.Background()); err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if repo.pinned != 0 {
		t.Fatalf("pin calls = %d, want 0 for already pinned upload", repo.pinned)
	}
	if object.putKey != key {
		t.Fatalf("put key = %q, want pinned key %q", object.putKey, key)
	}
}

func TestIngestionRetriesThenFails(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.wav")
	recording := sqlc.Recording{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		SourcePath:     &missing,
		Status:         "uploading",
	}
	repo := &fakeIngestionRepository{items: []sqlc.Recording{recording}}
	config := DefaultIngestionConfig(root)
	config.MaxAttempts = 2
	job, err := NewIngestionJob(
		repo,
		NewObjectStorage(&fakeObjectStore{}),
		config,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Ingest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repo.retries != 1 {
		t.Fatalf("retries = %d, want 1", repo.retries)
	}
	repo.items[0].UploadAttempts = 1
	if err := job.Ingest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repo.failed != 1 {
		t.Fatalf("failed = %d, want 1", repo.failed)
	}
}

func TestIngestionRejectsPathOutsideStaging(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside.wav")
	recording := sqlc.Recording{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		SourcePath:     &outside,
		Status:         "uploading",
	}
	repo := &fakeIngestionRepository{items: []sqlc.Recording{recording}}
	job, err := NewIngestionJob(
		repo,
		NewObjectStorage(&fakeObjectStore{}),
		DefaultIngestionConfig(root),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Ingest(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repo.retries != 1 {
		t.Fatalf("retries = %d, want 1", repo.retries)
	}
}
