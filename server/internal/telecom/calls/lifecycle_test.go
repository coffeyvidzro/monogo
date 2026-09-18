package calls

import "testing"

func TestLifecycleAlreadyApplied(t *testing.T) {
	tests := []struct {
		name      string
		snapshot  LifecycleSnapshot
		eventType LifecycleEventType
		want      bool
	}{
		{
			name:      "initiated is idempotent",
			snapshot:  LifecycleSnapshot{State: string(StateInitiating)},
			eventType: LifecycleInitiated,
			want:      true,
		},
		{
			name:      "answered already applied",
			snapshot:  LifecycleSnapshot{State: string(StateAnswered)},
			eventType: LifecycleAnswered,
			want:      true,
		},
		{
			name:      "answered not yet applied",
			snapshot:  LifecycleSnapshot{State: string(StateRinging)},
			eventType: LifecycleAnswered,
			want:      false,
		},
		{
			name: "held already applied",
			snapshot: LifecycleSnapshot{
				State:      string(StateActive),
				MediaState: string(MediaStateHeld),
			},
			eventType: LifecycleHeld,
			want:      true,
		},
		{
			name:      "terminal ignores later answer",
			snapshot:  LifecycleSnapshot{State: string(StateCompleted)},
			eventType: LifecycleAnswered,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lifecycleAlreadyApplied(tt.snapshot, tt.eventType); got != tt.want {
				t.Fatalf("lifecycleAlreadyApplied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTerminalLifecycleAndState(t *testing.T) {
	for _, eventType := range []LifecycleEventType{
		LifecycleCompleted,
		LifecycleFailed,
		LifecycleCancelled,
	} {
		if !isTerminalLifecycle(eventType) {
			t.Fatalf("%q should be terminal lifecycle", eventType)
		}
	}

	for _, state := range []State{
		StateCompleted,
		StateFailed,
		StateCancelled,
	} {
		if !isTerminalState(string(state)) {
			t.Fatalf("%q should be terminal state", state)
		}
	}

	if isTerminalLifecycle(LifecycleAnswered) {
		t.Fatal("answered must not be terminal lifecycle")
	}
	if isTerminalState(string(StateActive)) {
		t.Fatal("active must not be terminal state")
	}
}
