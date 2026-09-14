package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeLeaseCoordinator struct {
	allowed  bool
	reason   string
	err      error
	prefix   string
	bound    string
	released string
}

func (f *fakeLeaseCoordinator) AcquireCallLease(_ context.Context, prefix, _ string, _, _ int64, _ time.Duration) (bool, string, error) {
	f.prefix = prefix
	return f.allowed, f.reason, f.err
}
func (f *fakeLeaseCoordinator) BindCallLease(_ context.Context, _ string, _ string, callID string) error {
	f.bound = callID
	return f.err
}
func (f *fakeLeaseCoordinator) ReleaseCallLease(_ context.Context, _ string, id string) error {
	f.released = id
	return f.err
}
func (f *fakeLeaseCoordinator) RefreshCallLease(context.Context, string, string, time.Duration) error {
	return f.err
}

type fakeDailyUsage struct {
	seconds int64
	err     error
}

func (f *fakeDailyUsage) CarrierDailySeconds(context.Context, uuid.UUID, time.Time) (int64, error) {
	return f.seconds, f.err
}

func TestAdmissionRejectsDurableDailyLimit(t *testing.T) {
	leases := &fakeLeaseCoordinator{allowed: true}
	usage := &fakeDailyUsage{seconds: 60 * 10}
	controller, err := NewAdmissionController(leases, usage)
	if err != nil {
		t.Fatal(err)
	}
	limit := int64(10)
	_, err = controller.Admit(context.Background(), CallLimits{CarrierConnectionID: uuid.New(), MaxCPS: 1, MaxConcurrent: 1, MaxDailyMinutes: &limit})
	if !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("error = %v, want daily limit", err)
	}
	if leases.prefix != "" {
		t.Fatal("Redis lease acquired after durable daily limit was exhausted")
	}
}

func TestAdmissionMapsSharedLimitReason(t *testing.T) {
	for _, test := range []struct {
		reason string
		want   error
	}{{"cps", ErrCPSLimit}, {"concurrent", ErrConcurrentLimit}} {
		t.Run(test.reason, func(t *testing.T) {
			controller, err := NewAdmissionController(&fakeLeaseCoordinator{reason: test.reason}, &fakeDailyUsage{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = controller.Admit(context.Background(), CallLimits{CarrierConnectionID: uuid.New(), MaxCPS: 2, MaxConcurrent: 3})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestAdmissionFailsClosedWhenCoordinatorFails(t *testing.T) {
	controller, err := NewAdmissionController(&fakeLeaseCoordinator{err: errors.New("redis unavailable")}, &fakeDailyUsage{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Admit(context.Background(), CallLimits{CarrierConnectionID: uuid.New(), MaxCPS: 2, MaxConcurrent: 3}); err == nil {
		t.Fatal("admission succeeded while Redis was unavailable")
	}
}
