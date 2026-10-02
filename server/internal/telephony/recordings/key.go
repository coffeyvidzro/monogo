package recordings

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

func recordingStoragePrefix(organizationID uuid.UUID) (string, error) {
	if organizationID == uuid.Nil {
		return "", fmt.Errorf("recording organization id is required")
	}
	return "organizations/" + organizationID.String() + "/recordings/", nil
}

func recordingObjectKey(
	organizationID uuid.UUID,
	recordingID uuid.UUID,
	occurredAt time.Time,
	extension string,
) (string, error) {
	prefix, err := recordingStoragePrefix(organizationID)
	if err != nil {
		return "", err
	}
	if recordingID == uuid.Nil {
		return "", fmt.Errorf("recording id is required")
	}
	extension = strings.TrimPrefix(strings.TrimSpace(extension), ".")
	if extension == "" {
		extension = "bin"
	}
	if strings.ContainsAny(extension, `/\\`) {
		return "", fmt.Errorf("recording extension is invalid")
	}

	return prefix + occurredAt.UTC().Format("2006/01/02") + "/" +
		recordingID.String() + "." + extension, nil
}

func validateManagedRecordingKey(organizationID uuid.UUID, key string) error {
	prefix, err := recordingStoragePrefix(organizationID)
	if err != nil {
		return err
	}
	if key == "" || path.IsAbs(key) || path.Clean(key) != key {
		return fmt.Errorf("managed recording storage key is invalid")
	}
	if !strings.HasPrefix(key, prefix) || key == prefix {
		return fmt.Errorf("managed recording storage key is outside organization prefix")
	}
	return nil
}
