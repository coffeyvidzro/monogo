package retention

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type cleanupRepositoryStub struct {
	policies []Policy

	recordingIDs    []uuid.UUID
	conversationIDs []uuid.UUID
	auditEventIDs   []uuid.UUID

	organizationID uuid.UUID
	before         time.Time

	deletedConversationIDs []uuid.UUID
	deletedAuditEventIDs    []uuid.UUID
}

func (s *cleanupRepositoryStub) ListEnabled(context.Context) ([]Policy, error) {
	return s.policies, nil
}
func (s *cleanupRepositoryStub) ListExpiredRecordings(_ context.Context, organizationID uuid.UUID, before time.Time, _ int32) ([]uuid.UUID, error) {
	s.organizationID = organizationID
	s.before = before
	ids := s.recordingIDs
	s.recordingIDs = nil
	return ids, nil
}
func (s *cleanupRepositoryStub) ListExpiredConversations(_ context.Context, organizationID uuid.UUID, before time.Time, _ int32) ([]uuid.UUID, error) {
	s.organizationID = organizationID
	s.before = before
	ids := s.conversationIDs
	s.conversationIDs = nil
	return ids, nil
}
func (s *cleanupRepositoryStub) DeleteConversation(_ context.Context, organizationID, id uuid.UUID) error {
	s.organizationID = organizationID
	s.deletedConversationIDs = append(s.deletedConversationIDs, id)
	return nil
}
func (s *cleanupRepositoryStub) ListExpiredAuditEvents(_ context.Context, organizationID uuid.UUID, before time.Time, _ int32) ([]uuid.UUID, error) {
	s.organizationID = organizationID
	s.before = before
	ids := s.auditEventIDs
	s.auditEventIDs = nil
	return ids, nil
}
func (s *cleanupRepositoryStub) DeleteAuditEvent(_ context.Context, organizationID, id uuid.UUID) error {
	s.organizationID = organizationID
	s.deletedAuditEventIDs = append(s.deletedAuditEventIDs, id)
	return nil
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
		policies: []Policy{{
			OrganizationID: organizationID,
			Resource:       ResourceRecordings,
			RetentionDays:  30,
			Enabled:        true,
		}},
		recordingIDs: []uuid.UUID{recordingID},
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
	assertCutoff(t, repo.before, now.AddDate(0, 0, -30))
}

func TestCleanupJobDeletesExpiredConversations(t *testing.T) {
	organizationID := uuid.New()
	conversationID := uuid.New()
	now := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	repo := &cleanupRepositoryStub{
		policies: []Policy{{
			OrganizationID: organizationID,
			Resource:       ResourceConversations,
			RetentionDays:  90,
			Enabled:        true,
		}},
		conversationIDs: []uuid.UUID{conversationID},
	}
	job, err := NewCleanupJob(repo, &recordingDeleterStub{}, DefaultCleanupJobConfig())
	if err != nil {
		t.Fatalf("NewCleanupJob() error = %v", err)
	}
	job.now = func() time.Time { return now }
	if err := job.runOnce(context.Background()); err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}
	if repo.organizationID != organizationID || len(repo.deletedConversationIDs) != 1 || repo.deletedConversationIDs[0] != conversationID {
		t.Fatalf("tenant-scoped conversation deletion = org %s, ids %v", repo.organizationID, repo.deletedConversationIDs)
	}
	assertCutoff(t, repo.before, now.AddDate(0, 0, -90))
}

func TestCleanupJobDeletesExpiredAuditEvents(t *testing.T) {
	organizationID := uuid.New()
	auditEventID := uuid.New()
	now := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	repo := &cleanupRepositoryStub{
		policies: []Policy{{
			OrganizationID: organizationID,
			Resource:       ResourceAuditEvents,
			RetentionDays:  365,
			Enabled:        true,
		}},
		auditEventIDs: []uuid.UUID{auditEventID},
	}
	job, err := NewCleanupJob(repo, &recordingDeleterStub{}, DefaultCleanupJobConfig())
	if err != nil {
		t.Fatalf("NewCleanupJob() error = %v", err)
	}
	job.now = func() time.Time { return now }
	if err := job.runOnce(context.Background()); err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}
	if repo.organizationID != organizationID || len(repo.deletedAuditEventIDs) != 1 || repo.deletedAuditEventIDs[0] != auditEventID {
		t.Fatalf("tenant-scoped audit deletion = org %s, ids %v", repo.organizationID, repo.deletedAuditEventIDs)
	}
	assertCutoff(t, repo.before, now.AddDate(0, 0, -365))
}

func assertCutoff(t *testing.T, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("cutoff = %s, want %s", got, want)
	}
}
