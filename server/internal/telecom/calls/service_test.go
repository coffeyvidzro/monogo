package calls

import (
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/google/uuid"
)

func TestAdmissionFailureReason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "cps", err: calling.ErrAdmissionCPS, want: "carrier_cps_limit"},
		{name: "concurrent", err: calling.ErrAdmissionConcurrent, want: "carrier_concurrent_limit"},
		{name: "daily", err: ErrAdmissionDailyMinutes, want: "carrier_daily_minutes_limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := admissionFailureReason(tt.err); got != tt.want {
				t.Fatalf("admissionFailureReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateExistingInbound(t *testing.T) {
	organizationID := uuid.New()
	applicationID := uuid.New()
	carrierID := uuid.New()

	req := InboundAdmissionRequest{
		OrganizationID:      organizationID,
		ApplicationID:       applicationID,
		CarrierConnectionID: carrierID,
		ToURI:               "+14155550100",
	}

	call := sqlc.Call{
		OrganizationID:      organizationID,
		ApplicationID:       &applicationID,
		CarrierConnectionID: &carrierID,
		Direction:           string(DirectionInbound),
		State:               string(StateRinging),
		ToUri:               req.ToURI,
	}

	if err := validateExistingInbound(call, req); err != nil {
		t.Fatalf("validateExistingInbound() unexpected error: %v", err)
	}

	call.State = string(StateCompleted)
	if err := validateExistingInbound(call, req); err == nil {
		t.Fatal("expected terminal inbound call conflict")
	}
}

func TestValidateExistingInboundRejectsCarrierMismatch(t *testing.T) {
	organizationID := uuid.New()
	applicationID := uuid.New()
	carrierID := uuid.New()
	otherCarrierID := uuid.New()

	req := InboundAdmissionRequest{
		OrganizationID:      organizationID,
		ApplicationID:       applicationID,
		CarrierConnectionID: carrierID,
		ToURI:               "+14155550100",
	}

	call := sqlc.Call{
		OrganizationID:      organizationID,
		ApplicationID:       &applicationID,
		CarrierConnectionID: &otherCarrierID,
		Direction:           string(DirectionInbound),
		State:               string(StateRinging),
		ToUri:               req.ToURI,
	}

	if err := validateExistingInbound(call, req); err == nil {
		t.Fatal("expected carrier attribution conflict")
	}
}
