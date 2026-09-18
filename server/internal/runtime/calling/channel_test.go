package calling

import (
	"context"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
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

func TestAdmissionLimiterValidatesArgumentsBeforeRedis(t *testing.T) {
	limiter := &AdmissionLimiter{}

	if err := limiter.Acquire(
		context.Background(),
		uuid.Nil,
		"",
		routing.Limits{MaxCPS: 1, MaxConcurrentCalls: 1},
	); err == nil {
		t.Fatal("expected invalid admission identity error")
	}

	if err := limiter.Bind(context.Background(), uuid.Nil, "", uuid.Nil); err == nil {
		t.Fatal("expected invalid admission bind error")
	}

	if err := limiter.Release(context.Background(), uuid.Nil, ""); err == nil {
		t.Fatal("expected invalid admission release error")
	}

	if err := limiter.Refresh(context.Background(), uuid.Nil, uuid.Nil); err == nil {
		t.Fatal("expected invalid admission refresh error")
	}
}
