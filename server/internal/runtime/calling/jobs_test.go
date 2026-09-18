package calling

import (
	"context"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
	"github.com/google/uuid"
)

type fakeCallReconciliationRepository struct {
	calls []calls.ActiveAdmissionCall
}

func (f *fakeCallReconciliationRepository) ListActiveForAdmissionReconciliation(
	context.Context,
) ([]calls.ActiveAdmissionCall, error) {
	return f.calls, nil
}

type fakeReconciliationAdmission struct {
	refreshed []uuid.UUID
}

func (f *fakeReconciliationAdmission) Refresh(
	_ context.Context,
	_ uuid.UUID,
	callID uuid.UUID,
) error {
	f.refreshed = append(f.refreshed, callID)
	return nil
}

func TestCallReconciliationRefreshesActiveCarrierLeases(t *testing.T) {
	carrierID := uuid.New()
	callID := uuid.New()
	repo := &fakeCallReconciliationRepository{
		calls: []calls.ActiveAdmissionCall{
			{ID: callID, CarrierConnectionID: carrierID},
		},
	}
	admission := &fakeReconciliationAdmission{}

	job, err := NewReconciliationJob(
		repo,
		admission,
		DefaultReconciliationJobConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admission.refreshed) != 1 || admission.refreshed[0] != callID {
		t.Fatalf("refreshed = %v, want [%s]", admission.refreshed, callID)
	}
}

func TestCallReconciliationRestartIsIdempotent(t *testing.T) {
	carrierID := uuid.New()
	callID := uuid.New()
	repo := &fakeCallReconciliationRepository{
		calls: []calls.ActiveAdmissionCall{
			{ID: callID, CarrierConnectionID: carrierID},
		},
	}
	admission := &fakeReconciliationAdmission{}

	job, err := NewReconciliationJob(
		repo,
		admission,
		DefaultReconciliationJobConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := job.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(admission.refreshed) != 2 {
		t.Fatalf("refresh count = %d, want 2", len(admission.refreshed))
	}
}
