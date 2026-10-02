package recordings

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type fakeObjectStore struct {
	putKey, deletedKey string
	putBody            []byte
	putErr             error
}

func (f *fakeObjectStore) Put(_ context.Context, key, _ string, reader io.Reader, _ int64) error {
	f.putKey = key
	f.putBody, _ = io.ReadAll(reader)
	return f.putErr
}
func (f *fakeObjectStore) PlaybackURL(context.Context, string) (string, time.Time, error) {
	return "https://storage.example/recording?signature=test", time.Now().Add(time.Minute), nil
}
func (f *fakeObjectStore) Delete(_ context.Context, key string) error { f.deletedKey = key; return nil }
func (f *fakeObjectStore) Bucket() string                             { return "recordings" }

func TestObjectStoragePlaybackAndDelete(t *testing.T) {
	client := &fakeObjectStore{}
	storage := NewObjectStorage(client)
	organizationID := uuid.New()
	key := "organizations/" + organizationID.String() + "/recordings/2026/10/02/id.wav"
	provider, bucket := "s3", "recordings"
	recording := sqlc.Recording{
		OrganizationID: organizationID,
		StorageKey:     &key,
		StorageProvider: &provider,
		StorageBucket:   &bucket,
	}
	url, expires, err := storage.PlaybackURL(context.Background(), recording)
	if err != nil {
		t.Fatalf("PlaybackURL() error = %v", err)
	}
	if url == "" || expires.IsZero() {
		t.Fatal("PlaybackURL() returned invalid result")
	}
	if err := storage.Delete(context.Background(), recording); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if client.deletedKey != key {
		t.Fatalf("deleted key = %q", client.deletedKey)
	}
}

func TestObjectStorageRejectsManagedCrossTenantKey(t *testing.T) {
	storage := NewObjectStorage(&fakeObjectStore{})
	organizationID := uuid.New()
	otherOrganizationID := uuid.New()
	key := "organizations/" + otherOrganizationID.String() + "/recordings/id.wav"
	provider, bucket := "s3", "recordings"
	recording := sqlc.Recording{
		OrganizationID: organizationID,
		StorageKey:     &key,
		StorageProvider: &provider,
		StorageBucket:   &bucket,
	}

	_, _, err := storage.PlaybackURL(context.Background(), recording)
	if err == nil || !strings.Contains(err.Error(), "outside organization prefix") {
		t.Fatalf("PlaybackURL() error = %v", err)
	}
	if err := storage.Delete(context.Background(), recording); err == nil ||
		!strings.Contains(err.Error(), "outside organization prefix") {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestObjectStorageRejectsNonS3Recording(t *testing.T) {
	storage := NewObjectStorage(&fakeObjectStore{})
	if _, _, err := storage.PlaybackURL(context.Background(), sqlc.Recording{}); err == nil {
		t.Fatal("expected playback error")
	}
	if err := storage.Delete(context.Background(), sqlc.Recording{}); err == nil {
		t.Fatal("expected delete error")
	}
}
