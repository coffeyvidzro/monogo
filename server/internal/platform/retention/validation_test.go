package retention

import "testing"

func TestValidateUpsert(t *testing.T) {
	if err := validateUpsert(UpsertRequest{
		RetentionDays: 30,
		Enabled:       true,
	}); err != nil {
		t.Fatalf("validateUpsert() error = %v", err)
	}
	if err := validateUpsert(UpsertRequest{
		RetentionDays: 0,
	}); err == nil {
		t.Fatal("validateUpsert(0) error = nil")
	}
	if err := validateResource("unknown"); err == nil {
		t.Fatal("validateResource(unknown) error = nil")
	}
}
