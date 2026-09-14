package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

type fakeCallReconciliationRepository struct {
	calls         []sqlc.Call
	listCalls     int
	completed     int
	failed        int
	lastReason    string
	updatedBefore time.Time
}

func (f *fakeCallReconciliationRepository) ListForReconciliation(
	_ context.Context,
	updatedBefore time.Time,
	_ int32,
) ([]sqlc.Call, error) {
	f.listCalls++
	f.updatedBefore = updatedBefore

	result := make([]sqlc.Call, 0, len(f.calls))
	for _, call := range f.calls {
		if isTerminal(call.State) {
			continue
		}
		result = append(result, call)
	}
	return result, nil
}

func (f *fakeCallReconciliationRepository) MarkCompleted(
	_ context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	f.completed++
	if reason != nil {
		f.lastReason = *reason
	}
	return f.setState(organizationID, id, string(StateCompleted))
}

func (f *fakeCallReconciliationRepository) MarkFailed(
	_ context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	f.failed++
	if reason != nil {
		f.lastReason = *reason
	}
	return f.setState(organizationID, id, string(StateFailed))
}

func (f *fakeCallReconciliationRepository) setState(
	organizationID uuid.UUID,
	id uuid.UUID,
	state string,
) (sqlc.Call, error) {
	for i := range f.calls {
		if f.calls[i].OrganizationID == organizationID && f.calls[i].ID == id {
			f.calls[i].State = state
			return f.calls[i], nil
		}
	}
	return sqlc.Call{}, errors.New("call not found")
}

type fakeChannelInventory struct {
	channels []freeswitch.Channel
	err      error
}

func (f fakeChannelInventory) Channels(context.Context) ([]freeswitch.Channel, error) {
	return f.channels, f.err
}

func TestReconciliationRestartIsIdempotent(t *testing.T) {
	organizationID := uuid.New()
	callID := uuid.New()
	channelID := uuid.NewString()
	repo := &fakeCallReconciliationRepository{
		calls: []sqlc.Call{{
			ID:             callID,
			OrganizationID: organizationID,
			State:          string(StateAnswered),
			SipCallID:      &channelID,
		}},
	}

	job, err := NewReconciliationJob(
		repo,
		fakeChannelInventory{},
		ReconciliationJobConfig{Grace: time.Minute, BatchSize: 10},
	)
	if err != nil {
		t.Fatalf("NewReconciliationJob() error = %v", err)
	}
	job.now = func() time.Time { return time.Unix(1_000, 0) }

	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatalf("restart Reconcile() error = %v", err)
	}

	if repo.completed != 1 {
		t.Fatalf("completed mutations = %d, want 1", repo.completed)
	}
	if repo.failed != 0 {
		t.Fatalf("failed mutations = %d, want 0", repo.failed)
	}
	if repo.lastReason != reconciliationHangupReason {
		t.Fatalf("hangup reason = %q, want %q", repo.lastReason, reconciliationHangupReason)
	}
}

func TestReconciliationChaosDoesNotMutateWithoutInventory(t *testing.T) {
	channelID := uuid.NewString()
	repo := &fakeCallReconciliationRepository{
		calls: []sqlc.Call{{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			State:          string(StateActive),
			SipCallID:      &channelID,
		}},
	}

	job, err := NewReconciliationJob(
		repo,
		fakeChannelInventory{err: errors.New("freeswitch unavailable")},
		DefaultReconciliationJobConfig(),
	)
	if err != nil {
		t.Fatalf("NewReconciliationJob() error = %v", err)
	}

	if err := job.Reconcile(context.Background()); err == nil {
		t.Fatal("Reconcile() error = nil, want inventory error")
	}
	if repo.listCalls != 0 {
		t.Fatalf("database reconciliation queries = %d, want 0", repo.listCalls)
	}
	if repo.completed != 0 || repo.failed != 0 {
		t.Fatalf("mutations = completed:%d failed:%d, want none", repo.completed, repo.failed)
	}
}

func TestReconciliationKeepsLiveChannel(t *testing.T) {
	channelID := uuid.NewString()
	repo := &fakeCallReconciliationRepository{
		calls: []sqlc.Call{{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			State:          string(StateActive),
			SipCallID:      &channelID,
		}},
	}

	job, err := NewReconciliationJob(
		repo,
		fakeChannelInventory{channels: []freeswitch.Channel{{UUID: channelID}}},
		DefaultReconciliationJobConfig(),
	)
	if err != nil {
		t.Fatalf("NewReconciliationJob() error = %v", err)
	}

	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if repo.completed != 0 || repo.failed != 0 {
		t.Fatalf("mutations = completed:%d failed:%d, want none", repo.completed, repo.failed)
	}
}
