package calls

import (
	"testing"
	"time"
)

func TestNewReconciliationJobDefaultsInterval(t *testing.T) {
	job, err := NewReconciliationJob(
		&Repository{},
		&Service{},
		ReconciliationJobConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if job.config.Interval != 30*time.Second {
		t.Fatalf("interval = %s, want 30s", job.config.Interval)
	}
}

func TestNewReconciliationJobRequiresDependencies(t *testing.T) {
	if _, err := NewReconciliationJob(nil, &Service{}, DefaultReconciliationJobConfig()); err == nil {
		t.Fatal("expected repository validation error")
	}
	if _, err := NewReconciliationJob(&Repository{}, nil, DefaultReconciliationJobConfig()); err == nil {
		t.Fatal("expected service validation error")
	}
}
