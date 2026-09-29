package messaging

import (
	"testing"

	"github.com/google/uuid"
)

func TestManagedMessageOperationIDIsStablePerMessage(t *testing.T) {
	messageID := uuid.New()
	first := managedMessageOperationID(messageID)
	second := managedMessageOperationID(messageID)
	other := managedMessageOperationID(uuid.New())

	if first != second {
		t.Fatalf("replayed operation id = %s, want %s", second, first)
	}
	if first == other {
		t.Fatal("different messages produced the same billing operation id")
	}
}
