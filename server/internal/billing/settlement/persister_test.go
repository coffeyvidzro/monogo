package settlement

import (
	"math"
	"testing"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
)

func TestValidateWalletProjection(t *testing.T) {
	t.Parallel()

	event := redisintegration.OCSEvent{
		WalletID:            uuid.New(),
		WalletVersion:       3,
		BalanceDeltaMicros:  -200,
		ReservedDeltaMicros: -200,
		BalanceAfterMicros:  800,
		ReservedAfterMicros: 100,
	}
	if err := validateWalletProjection(2, 1_000, 300, event); err != nil {
		t.Fatalf(
			"validate projection: %v",
			err,
		)
	}
}

func TestValidateWalletProjectionDetectsVersionGap(t *testing.T) {
	t.Parallel()

	event := redisintegration.OCSEvent{
		WalletID:            uuid.New(),
		WalletVersion:       5,
		BalanceAfterMicros:  1_000,
		ReservedAfterMicros: 0,
	}
	if err := validateWalletProjection(2, 1_000, 0, event); err == nil {
		t.Fatal("expected wallet version gap")
	}
}

func TestSafeDeltaMatchesRejectsOverflow(t *testing.T) {
	t.Parallel()

	if safeDeltaMatches(math.MaxInt64, 1, math.MinInt64) {
		t.Fatal("expected positive overflow to fail")
	}
	if safeDeltaMatches(math.MinInt64, -1, math.MaxInt64) {
		t.Fatal("expected negative overflow to fail")
	}
}
