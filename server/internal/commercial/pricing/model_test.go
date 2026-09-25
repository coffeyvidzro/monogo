package pricing

import (
	"math"
	"testing"
)

func TestQuoteFromWholesale(t *testing.T) {
	quote, err := QuoteFromWholesale(
		1_000_000,
		2_000,
		60,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if quote.CustomerRateMicros != 1_200_000 {
		t.Fatalf("customer rate = %d, want 1200000", quote.CustomerRateMicros)
	}
	quote.CustomerRateMicros--
	if err := quote.Validate(); err == nil {
		t.Fatal("mismatched customer rate accepted")
	}
}

func TestQuoteFromWholesaleRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		wholesale int64
		markup    int64
		increment int64
		minimum   int64
	}{
		{
			name:      "zero wholesale rate",
			markup:    2_000,
			increment: 60,
		},
		{
			name:      "zero markup",
			wholesale: 1_000,
			increment: 60,
		},
		{
			name:      "negative markup",
			wholesale: 1_000,
			markup:    -1,
			increment: 60,
		},
		{
			name:      "invalid increment",
			wholesale: 1_000,
			markup:    2_000,
		},
		{
			name:      "negative minimum",
			wholesale: 1_000,
			markup:    2_000,
			increment: 60,
			minimum:   -1,
		},
		{
			name:      "customer rate overflow",
			wholesale: math.MaxInt64,
			markup:    2_000,
			increment: 60,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := QuoteFromWholesale(
				test.wholesale,
				test.markup,
				test.increment,
				test.minimum,
			)
			if err == nil {
				t.Fatal("invalid pricing policy was accepted")
			}
		})
	}
}

func TestQuoteRatesActualUsageInsteadOfReservedDuration(t *testing.T) {
	quote, err := QuoteFromWholesale(
		1_000_000,
		2_000,
		60,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		seconds  int64
		want     int64
		billable int64
	}{
		{
			name:    "verified zero usage",
			seconds: 0,
			want:    0,
		},
		{
			name:     "one second bills one minute",
			seconds:  1,
			want:     120,
			billable: 60,
		},
		{
			name:     "61 seconds bills two minutes",
			seconds:  61,
			want:     240,
			billable: 120,
		},
		{
			name:     "exact minute remains one minute",
			seconds:  60,
			want:     120,
			billable: 60,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rating, err := quote.Rate(test.seconds)
			if err != nil {
				t.Fatal(err)
			}
			if rating.CustomerAmountMinor != test.want ||
				rating.BillableSeconds != test.billable {
				t.Fatalf("rating = %+v, want amount %d and %d seconds",
					rating, test.want, test.billable)
			}
		})
	}
}

func TestQuoteHonorsMinimumAndFractionalCentRounding(t *testing.T) {
	quote, err := QuoteFromWholesale(
		1_000,
		2_000,
		1,
		30,
	)
	if err != nil {
		t.Fatal(err)
	}
	rating, err := quote.Rate(1)
	if err != nil {
		t.Fatal(err)
	}
	if rating.BillableSeconds != 30 || rating.CustomerAmountMinor != 1 {
		t.Fatalf("unexpected minimum-duration rating: %+v", rating)
	}
}

func TestQuoteRejectsInvalidUsageAndLargeRates(t *testing.T) {
	quote, err := QuoteFromWholesale(
		1_000_000,
		2_000,
		60,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, seconds := range []int64{-1, 86_401} {
		if _, err := quote.Rate(seconds); err == nil {
			t.Fatalf("invalid %d seconds accepted", seconds)
		}
	}
}
