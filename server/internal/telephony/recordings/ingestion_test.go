package recordings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type fakeIngestionRepository struct {
	items                      []sqlc.Recording
	completed, retries, failed int
}

func (f *fakeIngestionRepository) ListForUpload(context.Context, time.Time, time.Time, int32) ([]sqlc.Recording, error) {
	return f.items, nil
}
func (f *fakeIngestionRepository) CompleteUpload(_ context.Context, item sqlc.Recording, key, provider, bucket, format string, size int64) (sqlc.Recording, error) {
	f.completed++
	item.StorageKey, item.StorageProvider, item.StorageBucket, item.Format, item.FileSizeBytes = &key, &provider, &bucket, &format, &size
	item.Status = string(StatusCompleted)
	return item, nil
}
func (f *fakeIngestionRepository) RetryUpload(context.Context, sqlc.Recording, time.Time, string) error {
	f.retries++
	return nil
}
func (f *fakeIngestionRepository) Fail(_ context.Context, item sqlc.Recording) (sqlc.Recording, error) {
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
	recording := sqlc.Recording{ID: uuid.New(), OrganizationID: uuid.New(), SourcePath: &path, Status: "uploading"}
	repo, object := &fakeIngestionRepository{items: []sqlc.Recording{recording}}, &fakeObjectStore{}
	job, err := NewIngestionJob(repo, NewObjectStorage(object), DefaultIngestionConfig(root))
	if err != nil {
		t.Fatal(err)
	}
	job.now = func() time.Time { return time.Date(2026, 9, 20, 1, 2, 3, 0, time.UTC) }
	if err := job.Ingest(context.Background()); err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}
	if repo.completed != 1 || string(object.putBody) != "RIFF-recording" {
		t.Fatalf("completed=%d body=%q", repo.completed, object.putBody)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged file still exists: %v", err)
	}
}

func TestIngestionRetriesThenFails(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.wav")
	recording := sqlc.Recording{ID: uuid.New(), OrganizationID: uuid.New(), SourcePath: &missing, Status: "uploading"}
	repo := &fakeIngestionRepository{items: []sqlc.Recording{recording}}
	config := DefaultIngestionConfig(root)
	config.MaxAttempts = 2
	job, err := NewIngestionJob(repo, NewObjectStorage(&fakeObjectStore{}), config)
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
	recording := sqlc.Recording{ID: uuid.New(), OrganizationID: uuid.New(), SourcePath: &outside, Status: "uploading"}
	repo := &fakeIngestionRepository{items: []sqlc.Recording{recording}}
	job, err := NewIngestionJob(repo, NewObjectStorage(&fakeObjectStore{}), DefaultIngestionConfig(root))
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
