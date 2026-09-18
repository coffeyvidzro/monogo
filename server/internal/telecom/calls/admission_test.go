package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
)

type fakeCallLeaseStore struct {
	allowed       bool
	reason        string
	acquireErr    error
	acquireCalls  int
	boundLeaseID  string
	boundCallID   string
	releasedID    string
	refreshedID   string
}

func (s *fakeCallLeaseStore) AcquireCallLease(
	_ context.Context,
	_ string,
	_ string,
	_, _ int64,
	_ time.Duration,
) (bool, string, error) {
	s.acquireCalls++
	return s.allowed, s.reason, s.acquireErr
}

func (s *fakeCallLeaseStore) BindCallLease(
	_ context.Context,
	_ string,
	leaseID, callID string,
) error {
	s.boundLeaseID = leaseID
	s.boundCallID = callID
	return nil
}

func (s *fakeCallLeaseStore) ReleaseCallLease(
	_ context.Context,
	_ string,
	id string,
) error {
	s.releasedID = id
	return nil
}

func (s *fakeCallLeaseStore) RefreshCallLease(
	_ context.Context,
	_ string,
	id string,
	_ time.Duration,
) error {
	s.refreshedID = id
	return nil
}

type fakeDailyUsageReader struct {
	seconds int64
	err     error
}

func (r fakeDailyUsageReader) CarrierDailyUsageSeconds(
	context.Context,
	uuid.UUID,
) (int64, error) {
	return r.seconds, r.err
}

func TestAdmissionLimiterRejectsDailyMinuteLimitBeforeLease(t *testing.T) {
	store := &fakeCallLeaseStore{allowed: true}
	limiter := &RedisAdmissionLimiter{
		store: store,
		usage: fakeDailyUsageReader{seconds: 60 * 100},
	}

	err := limiter.Acquire(
		context.Background(),
		uuid.New(),
		"lease-1",
		routing.Limits{
			MaxCPS:             10,
			MaxConcurrentCalls: 20,
			MaxDailyMinutes:    int64Ptr(100),
		},
	)
	if !errors.Is(err, ErrAdmissionDailyMinutes) {
		t.Fatalf("error = %v, want ErrAdmissionDailyMinutes", err)
	}
	if store.acquireCalls != 0 {
		t.Fatalf("acquire calls = %d, want 0", store.acquireCalls)
	}
}

func TestAdmissionLimiterMapsCPSRejection(t *testing.T) {
	store := &fakeCallLeaseStore{reason: "cps"}
	limiter := &RedisAdmissionLimiter{
		store: store,
		usage: fakeDailyUsageReader{},
	}

	err := limiter.Acquire(
		context.Background(),
		uuid.New(),
		"lease-1",
		routing.Limits{MaxCPS: 1, MaxConcurrentCalls: 10},
	)
	if !errors.Is(err, ErrAdmissionCPS) {
		t.Fatalf("error = %v, want ErrAdmissionCPS", err)
	}
}

func TestAdmissionLimiterMapsConcurrentRejection(t *testing.T) {
	store := &fakeCallLeaseStore{reason: "concurrent"}
	limiter := &RedisAdmissionLimiter{
		store: store,
		usage: fakeDailyUsageReader{},
	}

	err := limiter.Acquire(
		context.Background(),
		uuid.New(),
		"lease-1",
		routing.Limits{MaxCPS: 10, MaxConcurrentCalls: 1},
	)
	if !errors.Is(err, ErrAdmissionConcurrent) {
		t.Fatalf("error = %v, want ErrAdmissionConcurrent", err)
	}
}

func TestAdmissionLimiterAllowsWithinLimits(t *testing.T) {
	store := &fakeCallLeaseStore{allowed: true, reason: "ok"}
	limiter := &RedisAdmissionLimiter{
		store: store,
		usage: fakeDailyUsageReader{seconds: 59},
	}

	err := limiter.Acquire(
		context.Background(),
		uuid.New(),
		"lease-1",
		routing.Limits{
			MaxCPS:             10,
			MaxConcurrentCalls: 20,
			MaxDailyMinutes:    int64Ptr(1),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if store.acquireCalls != 1 {
		t.Fatalf("acquire calls = %d, want 1", store.acquireCalls)
	}
}

func TestAdmissionLimiterLeaseLifecycle(t *testing.T) {
	store := &fakeCallLeaseStore{allowed: true}
	limiter := &RedisAdmissionLimiter{
		store: store,
		usage: fakeDailyUsageReader{},
	}
	carrierID := uuid.New()
	callID := uuid.New()

	if err := limiter.Bind(context.Background(), carrierID, "channel-1", callID); err != nil {
		t.Fatal(err)
	}
	if store.boundLeaseID != "channel-1" || store.boundCallID != callID.String() {
		t.Fatalf("unexpected bind: lease=%q call=%q", store.boundLeaseID, store.boundCallID)
	}

	if err := limiter.Refresh(context.Background(), carrierID, callID); err != nil {
		t.Fatal(err)
	}
	if store.refreshedID != callID.String() {
		t.Fatalf("refreshed id = %q", store.refreshedID)
	}

	if err := limiter.Release(context.Background(), carrierID, callID.String()); err != nil {
		t.Fatal(err)
	}
	if store.releasedID != callID.String() {
		t.Fatalf("released id = %q", store.releasedID)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
