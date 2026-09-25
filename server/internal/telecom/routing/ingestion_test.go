package routing

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateRateObservation(t *testing.T) {
	base := RateObservation{
		CarrierConnectionID: uuid.New(),
		DestinationPrefix:   "233",
		Currency:            "USD",
		RateMicros:          1250,
		EffectiveAt:         time.Now().UTC(),
	}
	if err := validateRateObservation(base); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		observation RateObservation
	}{
		{
			name: "missing connection",
			observation: func() RateObservation {
				invalid := base
				invalid.CarrierConnectionID = uuid.Nil
				return invalid
			}(),
		},
		{
			name: "non-digit prefix",
			observation: func() RateObservation {
				invalid := base
				invalid.DestinationPrefix = "233+"
				return invalid
			}(),
		},
		{
			name: "unconverted currency",
			observation: func() RateObservation {
				invalid := base
				invalid.Currency = "GHS"
				return invalid
			}(),
		},
		{
			name: "negative rate",
			observation: func() RateObservation {
				invalid := base
				invalid.RateMicros = -1
				return invalid
			}(),
		},
		{
			name: "invalid expiry",
			observation: func() RateObservation {
				invalid := base
				expiry := invalid.EffectiveAt.Add(-time.Second)
				invalid.ExpiresAt = &expiry
				return invalid
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateRateObservation(test.observation); err == nil {
				t.Fatal("invalid rate observation was accepted")
			}
		})
	}
}

func TestValidateHealthObservation(t *testing.T) {
	now := time.Now().UTC()
	base := HealthObservation{
		EndpointID:           uuid.New(),
		ASRBasisPoints:        9_000,
		ALOCMilliseconds:      80_000,
		LatencyMilliseconds:   20,
		PacketLossBasisPoints: 100,
		SampleCount:           10,
		ObservedAt:            now,
	}
	if err := validateHealthObservation(base, now); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		observation HealthObservation
	}{
		{
			name: "no measured samples",
			observation: func() HealthObservation {
				invalid := base
				invalid.SampleCount = 0
				return invalid
			}(),
		},
		{
			name: "stale observation",
			observation: func() HealthObservation {
				invalid := base
				invalid.ObservedAt = now.Add(-2 * time.Minute)
				return invalid
			}(),
		},
		{
			name: "future observation",
			observation: func() HealthObservation {
				invalid := base
				invalid.ObservedAt = now.Add(time.Second)
				return invalid
			}(),
		},
		{
			name: "impossible ASR",
			observation: func() HealthObservation {
				invalid := base
				invalid.ASRBasisPoints = 10_001
				return invalid
			}(),
		},
		{
			name: "negative latency",
			observation: func() HealthObservation {
				invalid := base
				invalid.LatencyMilliseconds = -1
				return invalid
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateHealthObservation(test.observation, now); err == nil {
				t.Fatal("invalid health observation was accepted")
			}
		})
	}
}
