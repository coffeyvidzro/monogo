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
	voiceAgentID := uuid.New()
	trunkID := uuid.New()

	req := InboundAdmissionRequest{
		OrganizationID:      organizationID,
		VoiceAgentID:        voiceAgentID,
		TrunkID: trunkID,
		ToURI:               "+14155550100",
	}

	call := sqlc.Call{
		OrganizationID:      organizationID,
		VoiceAgentID:        &voiceAgentID,
		TrunkID: &trunkID,
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
	voiceAgentID := uuid.New()
	trunkID := uuid.New()
	otherCarrierID := uuid.New()

	req := InboundAdmissionRequest{
		OrganizationID:      organizationID,
		VoiceAgentID:        voiceAgentID,
		TrunkID: trunkID,
		ToURI:               "+14155550100",
	}

	call := sqlc.Call{
		OrganizationID:      organizationID,
		VoiceAgentID:        &voiceAgentID,
		TrunkID: &otherCarrierID,
		Direction:           string(DirectionInbound),
		State:               string(StateRinging),
		ToUri:               req.ToURI,
	}

	if err := validateExistingInbound(call, req); err == nil {
		t.Fatal("expected carrier attribution conflict")
	}
}
