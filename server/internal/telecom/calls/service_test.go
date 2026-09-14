package calls

import (
	"testing"

	"github.com/leamout/leamout/internal/database/sqlc"
)

func TestCanAnswer(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{state: "initiating", want: true},
		{state: "ringing", want: true},
		{state: "answered", want: false},
		{state: "active", want: false},
		{state: "completed", want: false},
		{state: "failed", want: false},
		{state: "cancelled", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			if got := canAnswer(tt.state); got != tt.want {
				t.Fatalf("canAnswer(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestIsAnswerIdempotent(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{state: "initiating", want: false},
		{state: "ringing", want: false},
		{state: "answered", want: true},
		{state: "active", want: true},
		{state: "completed", want: false},
		{state: "failed", want: false},
		{state: "cancelled", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			if got := isAnswerIdempotent(tt.state); got != tt.want {
				t.Fatalf("isAnswerIdempotent(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{state: "initiating", want: false},
		{state: "ringing", want: false},
		{state: "answered", want: false},
		{state: "active", want: false},
		{state: "completed", want: true},
		{state: "failed", want: true},
		{state: "cancelled", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			if got := isTerminal(tt.state); got != tt.want {
				t.Fatalf("isTerminal(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestValidateControl(t *testing.T) {
	connectedActions := []controlAction{
		controlTransfer,
		controlHold,
		controlUnhold,
		controlPlay,
		controlStop,
		controlRecord,
		controlDTMF,
	}

	for _, action := range connectedActions {
		for _, state := range []string{"answered", "active"} {
			t.Run(string(action)+"/allows-"+state, func(t *testing.T) {
				call := sqlc.Call{Direction: "outbound", State: state}
				if err := validateControl(call, action); err != nil {
					t.Fatalf("validateControl(%q, %q) returned error: %v", state, action, err)
				}
			})
		}

		for _, state := range []string{"initiating", "ringing", "completed", "failed", "cancelled"} {
			t.Run(string(action)+"/rejects-"+state, func(t *testing.T) {
				call := sqlc.Call{Direction: "outbound", State: state}
				if err := validateControl(call, action); err == nil {
					t.Fatalf("validateControl(%q, %q) expected conflict", state, action)
				}
			})
		}
	}
}

func TestValidateAnswerControl(t *testing.T) {
	for _, state := range []string{"initiating", "ringing", "answered", "active"} {
		t.Run("inbound/"+state, func(t *testing.T) {
			call := sqlc.Call{Direction: "inbound", State: state}
			if err := validateControl(call, controlAnswer); err != nil {
				t.Fatalf("validateControl(%q, answer) returned error: %v", state, err)
			}
		})
	}

	for _, state := range []string{"completed", "failed", "cancelled"} {
		t.Run("inbound/rejects-"+state, func(t *testing.T) {
			call := sqlc.Call{Direction: "inbound", State: state}
			if err := validateControl(call, controlAnswer); err == nil {
				t.Fatalf("validateControl(%q, answer) expected conflict", state)
			}
		})
	}

	for _, state := range []string{"initiating", "ringing", "answered", "active"} {
		t.Run("outbound/rejects-"+state, func(t *testing.T) {
			call := sqlc.Call{Direction: "outbound", State: state}
			if err := validateControl(call, controlAnswer); err == nil {
				t.Fatalf("outbound answer in %q expected conflict", state)
			}
		})
	}
}

func TestHangupStateClassification(t *testing.T) {
	for _, state := range []string{"initiating", "ringing"} {
		if !isPreAnswer(state) {
			t.Fatalf("isPreAnswer(%q) = false, want true", state)
		}
	}
	for _, state := range []string{"answered", "active"} {
		if !isConnected(state) {
			t.Fatalf("isConnected(%q) = false, want true", state)
		}
	}
}

func TestMediaStateClassification(t *testing.T) {
	held := sqlc.Call{MediaState: string(MediaStateHeld)}
	if !isHeld(held) || isMediaActive(held) {
		t.Fatal("held call media state was classified incorrectly")
	}
	if err := validateMediaState(held); err != nil {
		t.Fatalf("held media state returned error: %v", err)
	}

	active := sqlc.Call{MediaState: string(MediaStateActive)}
	if isHeld(active) || !isMediaActive(active) {
		t.Fatal("active call media state was classified incorrectly")
	}
	if err := validateMediaState(active); err != nil {
		t.Fatalf("active media state returned error: %v", err)
	}

	invalid := sqlc.Call{MediaState: "unknown"}
	if err := validateMediaState(invalid); err == nil {
		t.Fatal("invalid media state expected conflict")
	}
}

func TestStatusForState(t *testing.T) {
	if got := statusForState(string(StateInitiating)); got != StatusInitiated {
		t.Fatalf("initiating state mapped to %q, want %q", got, StatusInitiated)
	}
	if got := statusForState(string(StateAnswered)); got != StatusAnswered {
		t.Fatalf("answered state mapped to %q, want %q", got, StatusAnswered)
	}
}
