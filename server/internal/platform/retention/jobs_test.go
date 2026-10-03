package retention

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type cleanupRepositoryStub struct {
	policies       []Policy
	ids            []uuid.UUID
	organizationID uuid.UUID
	before         time.Time
}

func (s *cleanupRepositoryStub) ListEnabled(context.Context) ([]Policy, error) {
	return s.policies, nil
}
func (s *cleanupRepositoryStub) ListExpiredRecordings(_ context.Context, organizationID uuid.UUID, before time.Time, _ int32) ([]uuid.UUID, error) {
	s.organizationID = organizationID
	s.before = before
	ids := s.ids
	s.ids = nil
	return ids, nil
}

type recordingDeleterStub struct {
	organizationID uuid.UUID
	ids            []uuid.UUID
}

func (s *recordingDeleterStub) Delete(_ context.Context, organizationID, id uuid.UUID) error {
	s.organizationID = organizationID
	s.ids = append(s.ids, id)
	return nil
}
func TestCleanupJobDeletesExpiredRecordingThroughStorageAwareService(t *testing.T) {
	organizationID := uuid.New()
	recordingID := uuid.New()
	now := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	repo := &cleanupRepositoryStub{
		policies: []Policy{
			{
				OrganizationID: organizationID,
				Resource:       ResourceRecordings,
				RetentionDays:  30,
				Enabled:        true,
			},
		},
		ids: []uuid.UUID{
			recordingID,
		},
	}
	deleter := &recordingDeleterStub{}
	job, err := NewCleanupJob(repo, deleter, DefaultCleanupJobConfig())
	if err != nil {
		t.Fatalf("NewCleanupJob() error = %v", err)
	}
	job.now = func() time.Time { return now }
	if err := job.runOnce(context.Background()); err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}
	if repo.organizationID != organizationID || deleter.organizationID != organizationID || len(deleter.ids) != 1 || deleter.ids[0] != recordingID {
		t.Fatalf("tenant-scoped deletion = repo %s, deleter %s, ids %v", repo.organizationID, deleter.organizationID, deleter.ids)
	}
	wantCutoff := now.AddDate(0, 0, -30)
	if !repo.before.Equal(wantCutoff) {
		t.Fatalf("cutoff = %s, want %s", repo.before, wantCutoff)
	}
}
