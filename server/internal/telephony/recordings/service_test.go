package recordings

import (
	"context"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type fakeServiceRepository struct {
	recording sqlc.Recording
	deleted   int
}

func (f *fakeServiceRepository) Get(context.Context, uuid.UUID, uuid.UUID) (sqlc.Recording, error) {
	return f.recording, nil
}
func (f *fakeServiceRepository) GetIncludingDeleted(context.Context, uuid.UUID, uuid.UUID) (sqlc.Recording, error) {
	return f.recording, nil
}
func (f *fakeServiceRepository) GetByCallStorageKey(context.Context, uuid.UUID, string) (sqlc.Recording, error) {
	return f.recording, nil
}
func (f *fakeServiceRepository) GetCallOrganizationID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.recording.OrganizationID, nil
}
func (f *fakeServiceRepository) List(context.Context, uuid.UUID, int32, int32) ([]sqlc.Recording, error) {
	return []sqlc.Recording{f.recording}, nil
}
func (f *fakeServiceRepository) Start(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (sqlc.Recording, error) {
	return f.recording, nil
}
func (f *fakeServiceRepository) MarkReadyForUpload(context.Context, sqlc.Recording, time.Time) (sqlc.Recording, error) {
	return f.recording, nil
}
func (f *fakeServiceRepository) Delete(_ context.Context, item sqlc.Recording) (sqlc.Recording, error) {
	f.deleted++
	item.Status = string(StatusDeleted)
	return item, nil
}

func TestServicePlaybackReturnsSignedURL(t *testing.T) {
	organizationID, recordingID := uuid.New(), uuid.New()
	key, provider, bucket := "recordings/org/id.wav", "s3", "recordings"
	repo := &fakeServiceRepository{recording: sqlc.Recording{ID: recordingID, OrganizationID: organizationID, Status: string(StatusCompleted), StorageKey: &key, StorageProvider: &provider, StorageBucket: &bucket}}
	service := NewService(repo, NewObjectStorage(&fakeObjectStore{}))
	result, err := service.Playback(context.Background(), organizationID, recordingID)
	if err != nil {
		t.Fatalf("Playback() error = %v", err)
	}
	if result.URL == "" || result.ExpiresAt.IsZero() {
		t.Fatal("Playback() returned invalid signed URL")
	}
}

func TestServiceDeleteRemovesObjectBeforeRow(t *testing.T) {
	organizationID, recordingID := uuid.New(), uuid.New()
	key, provider, bucket := "recordings/org/id.wav", "s3", "recordings"
	repo := &fakeServiceRepository{recording: sqlc.Recording{ID: recordingID, OrganizationID: organizationID, Status: string(StatusCompleted), StorageKey: &key, StorageProvider: &provider, StorageBucket: &bucket}}
	objects := &fakeObjectStore{}
	service := NewService(repo, NewObjectStorage(objects))
	if err := service.Delete(context.Background(), organizationID, recordingID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if objects.deletedKey != key || repo.deleted != 1 {
		t.Fatalf("object=%q row deletes=%d", objects.deletedKey, repo.deleted)
	}
}

func TestServiceRejectsDeleteDuringUpload(t *testing.T) {
	organizationID, recordingID := uuid.New(), uuid.New()
	repo := &fakeServiceRepository{recording: sqlc.Recording{ID: recordingID, OrganizationID: organizationID, Status: string(StatusUploading)}}
	service := NewService(repo, NewObjectStorage(&fakeObjectStore{}))
	if err := service.Delete(context.Background(), organizationID, recordingID); err == nil {
		t.Fatal("expected upload conflict")
	}
	if repo.deleted != 0 {
		t.Fatal("uploading row was deleted")
	}
}
