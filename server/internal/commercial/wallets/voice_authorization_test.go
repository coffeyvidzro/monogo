package wallets

import (
	"math"
	"testing"
)

func TestManagedCallReserveMinor(t *testing.T) {
	tests := []struct {
		name           string
		rateMicros     int64
		maximumMinutes int64
		want           int64
		wantError      bool
	}{
		{
			name:           "round up fractional cent",
			rateMicros:     1,
			maximumMinutes: 1,
			want:           1,
		},
		{
			name:           "integer dollar",
			rateMicros:     1_000_000,
			maximumMinutes: 5,
			want:           500,
		},
		{
			name:           "round up after duration multiplication",
			rateMicros:     3_333,
			maximumMinutes: 3,
			want:           1,
		},
		{
			name:           "round up nonintegral cent",
			rateMicros:     3_334,
			maximumMinutes: 3,
			want:           2,
		},
		{
			name:           "reject unpriced carrier",
			rateMicros:     0,
			maximumMinutes: 5,
			wantError:      true,
		},
		{
			name:           "reject zero duration",
			rateMicros:     500_000,
			maximumMinutes: 0,
			wantError:      true,
		},
		{
			name:           "reject negative duration",
			rateMicros:     500_000,
			maximumMinutes: -1,
			wantError:      true,
		},
		{
			name:           "reject integer overflow",
			rateMicros:     math.MaxInt64,
			maximumMinutes: 2,
			wantError:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ManagedCallReserveMinor(
				test.rateMicros,
				test.maximumMinutes,
			)
			if test.wantError {
				if err == nil {
					t.Fatal("expected an authorization error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("reserve minor = %d, want %d", got, test.want)
			}
		})
	}
}
