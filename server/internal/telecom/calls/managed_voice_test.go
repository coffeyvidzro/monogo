package calls

import (
	"testing"

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
