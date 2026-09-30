package subscribers

import "testing"

func TestSubscriberValidation(t *testing.T) {
	name, err := normalizeUsername(" alice-01 ")
	if err != nil || name != "alice-01" {
		t.Fatalf("normalizeUsername() = %q, %v", name, err)
	}
	if err := validatePassword("short"); err == nil {
		t.Fatal("accepted short password")
	}
	display := " Alice "
	got, err := normalizeDisplayName(&display)
	if err != nil || *got != "Alice" {
		t.Fatalf("normalizeDisplayName() = %v, %v", got, err)
	}
}
