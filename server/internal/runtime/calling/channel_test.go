package calling

import (
	"testing"

	"github.com/google/uuid"
)

func TestChannelKey(t *testing.T) {
	callID := uuid.New()
	want := "telecom:calls:channel:" + callID.String()
	if got := channelKey(callID); got != want {
		t.Fatalf("channel key = %q, want %q", got, want)
	}
}

func TestAdmissionPrefix(t *testing.T) {
	carrierID := uuid.New()
	want := "telecom:admission:carrier:" + carrierID.String()
	if got := admissionPrefix(carrierID); got != want {
		t.Fatalf("admission prefix = %q, want %q", got, want)
	}
}
