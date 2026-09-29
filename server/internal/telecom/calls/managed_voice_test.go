package calls

import (
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
)

func TestCommerciallyEligibleManagedRoutesRejectSupplierLoss(t *testing.T) {
	eligible := routing.OutboundRoute{
		CarrierConnectionID: uuid.New(),
		ProvisioningMode:    "managed",
		RateMicros:          20_000,
	}
	tooExpensive := routing.OutboundRoute{
		CarrierConnectionID: uuid.New(),
		ProvisioningMode:    "managed",
		RateMicros:          60_000,
	}
	byoc := routing.OutboundRoute{
		CarrierConnectionID: uuid.New(),
		ProvisioningMode:    "byoc",
		RateMicros:          0,
	}

	routes := commerciallyEligibleManagedRoutes(
		[]routing.OutboundRoute{
			tooExpensive,
			byoc,
			eligible,
		},
		50_000,
	)

	if len(routes) != 1 {
		t.Fatalf("eligible routes = %d, want 1", len(routes))
	}
	if routes[0].CarrierConnectionID != eligible.CarrierConnectionID {
		t.Fatalf(
			"eligible carrier = %s, want %s",
			routes[0].CarrierConnectionID,
			eligible.CarrierConnectionID,
		)
	}
}

func TestManagedVoiceCaptureAmountRoundsStartedMinutesAndReleasesUnusedFunds(t *testing.T) {
	answeredAt := time.Date(
		2026,
		time.September,
		28,
		12,
		0,
		0,
		0,
		time.UTC,
	)
	tests := []struct {
		name     string
		duration time.Duration
		want     int64
	}{
		{
			name:     "sub-minute call",
			duration: time.Second,
			want:     100_000,
		},
		{
			name:     "exact minute",
			duration: time.Minute,
			want:     100_000,
		},
		{
			name:     "started second minute",
			duration: time.Minute + time.Second,
			want:     200_000,
		},
		{
			name:     "authorization ceiling",
			duration: 11 * time.Minute,
			want:     1_000_000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			amount := managedVoiceCaptureAmount(
				1_000_000,
				answeredAt,
				answeredAt.Add(test.duration),
			)
			if amount != test.want {
				t.Fatalf("capture amount = %d, want %d", amount, test.want)
			}
		})
	}
}

func TestManagedVoiceOperationIDIsStablePerCall(t *testing.T) {
	callID := uuid.New()

	first := managedVoiceOperationID(callID)
	replayed := managedVoiceOperationID(callID)
	other := managedVoiceOperationID(uuid.New())

	if first != replayed {
		t.Fatalf(
			"operation id replay = %s, want %s",
			replayed,
			first,
		)
	}
	if first == other {
		t.Fatal("different calls produced the same managed voice operation id")
	}
}
