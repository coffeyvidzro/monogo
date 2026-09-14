package recordings

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type fakeRecordingReconciliationRepository struct {
	recordings []sqlc.Recording
	completed  int
}

func (f *fakeRecordingReconciliationRepository) ListForReconciliation(
	context.Context,
	time.Time,
	int32,
) ([]sqlc.Recording, error) {
	result := make([]sqlc.Recording, 0, len(f.recordings))
	for _, recording := range f.recordings {
		if recording.Status != string(StatusRecording) {
			continue
		}
		result = append(result, recording)
	}
	return result, nil
}

func (f *fakeRecordingReconciliationRepository) Complete(
	_ context.Context,
	recording sqlc.Recording,
) (sqlc.Recording, error) {
	f.completed++
	for i := range f.recordings {
		if f.recordings[i].ID == recording.ID {
			f.recordings[i].Status = string(StatusCompleted)
			return f.recordings[i], nil
		}
	}
	return recording, nil
}

func TestRecordingReconciliationRestartIsIdempotent(t *testing.T) {
	repo := &fakeRecordingReconciliationRepository{
		recordings: []sqlc.Recording{{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			CallID:         uuid.New(),
			Status:         string(StatusRecording),
		}},
	}

	job, err := NewReconciliationJob(
		repo,
		ReconciliationJobConfig{Grace: time.Minute, BatchSize: 10},
	)
	if err != nil {
		t.Fatalf("NewReconciliationJob() error = %v", err)
	}

	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatalf("restart Reconcile() error = %v", err)
	}

	if repo.completed != 1 {
		t.Fatalf("completed mutations = %d, want 1", repo.completed)
	}
}
