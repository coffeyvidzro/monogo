package authorization

import (
	"math"
	"testing"
)

func TestAmountForSecondsRoundsUpStartedMinutes(t *testing.T) {
	tests := []struct {
		name    string
		rate    int64
		seconds int64
		want    int64
	}{
		{
			name:    "zero usage",
			rate:    25,
			seconds: 0,
			want:    0,
		},
		{
			name:    "partial minute",
			rate:    25,
			seconds: 1,
			want:    25,
		},
		{
			name:    "whole minute",
			rate:    25,
			seconds: 60,
			want:    25,
		},
		{
			name:    "second minute",
			rate:    25,
			seconds: 61,
			want:    50,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := AmountForSeconds(test.rate, test.seconds)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("AmountForSeconds() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestAmountForSecondsRejectsInvalidAndOverflow(t *testing.T) {
	tests := []struct {
		rate    int64
		seconds int64
	}{
		{
			rate:    -1,
			seconds: 1,
		},
		{
			rate:    1,
			seconds: -1,
		},
		{
			rate:    math.MaxInt64,
			seconds: 61,
		},
	}

	for _, test := range tests {
		if _, err := AmountForSeconds(test.rate, test.seconds); err == nil {
			t.Fatalf("AmountForSeconds(%d, %d) succeeded", test.rate, test.seconds)
		}
	}
}
