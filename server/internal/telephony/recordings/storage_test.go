package recordings

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
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
	key, provider, bucket := "recordings/org/id.wav", "s3", "recordings"
	recording := sqlc.Recording{StorageKey: &key, StorageProvider: &provider, StorageBucket: &bucket}
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

func TestObjectStorageRejectsNonS3Recording(t *testing.T) {
	storage := NewObjectStorage(&fakeObjectStore{})
	if _, _, err := storage.PlaybackURL(context.Background(), sqlc.Recording{}); err == nil {
		t.Fatal("expected playback error")
	}
	if err := storage.Delete(context.Background(), sqlc.Recording{}); err == nil {
		t.Fatal("expected delete error")
	}
}
