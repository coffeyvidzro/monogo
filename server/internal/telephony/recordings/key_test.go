package recordings

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecordingObjectKeyUsesOrganizationNamespace(t *testing.T) {
	organizationID := uuid.New()
	recordingID := uuid.New()
	key, err := recordingObjectKey(
		organizationID,
		recordingID,
		time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC),
		"wav",
	)
	if err != nil {
		t.Fatalf("recordingObjectKey() error = %v", err)
	}
	want := "organizations/" + organizationID.String() +
		"/recordings/2026/10/02/" + recordingID.String() + ".wav"
	if key != want {
		t.Fatalf("recordingObjectKey() = %q, want %q", key, want)
	}
}

func TestValidateManagedRecordingKeyRejectsOtherOrganization(t *testing.T) {
	organizationID := uuid.New()
	otherOrganizationID := uuid.New()
	key := "organizations/" + otherOrganizationID.String() + "/recordings/file.wav"

	err := validateManagedRecordingKey(organizationID, key)
	if err == nil || !strings.Contains(err.Error(), "outside organization prefix") {
		t.Fatalf("validateManagedRecordingKey() error = %v", err)
	}
}

func TestValidateManagedRecordingKeyRejectsTraversal(t *testing.T) {
	organizationID := uuid.New()
	key := "organizations/" + organizationID.String() + "/recordings/../other/file.wav"

	if err := validateManagedRecordingKey(organizationID, key); err == nil {
		t.Fatal("validateManagedRecordingKey() error = nil")
	}
}
