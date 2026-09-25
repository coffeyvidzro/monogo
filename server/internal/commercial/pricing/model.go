// Package pricing owns deterministic customer tariffs for raw managed
// connectivity. Prepaid wallets hold and debit amounts; they do not set prices.
package pricing

import (
	"fmt"
	"math"
	"math/big"
)

// Quote is a decision-time wholesale-to-customer tariff for one carrier route.
// MarkupBasisPoints is a markup over wholesale cost, not a gross-margin rate.
// All rates are USD micros per minute, without floating-point arithmetic.
type Quote struct {
	Currency                string
	WholesaleRateMicros     int64
	CustomerRateMicros      int64
	MarkupBasisPoints       int64
	BillingIncrementSeconds int64
	MinimumBillableSeconds  int64
}

// Rating is the charge for actual usage under an already-quoted customer
// tariff. Estimated wholesale cost is not a substitute for the provider CDR.
type Rating struct {
	ActualSeconds           int64
	BillableSeconds         int64
	CustomerAmountMinor     int64
	EstimatedWholesaleMinor int64
}

// QuoteFromWholesale prices a verified wholesale rate using an approved
// markup and a customer-facing billing increment and minimum duration.
// The caller must validate source, currency, route, effective date, and
// commercial policy before quoting; this method does not fetch rate sheets.
func QuoteFromWholesale(
	wholesaleRateMicros int64,
	markupBasisPoints int64,
	billingIncrementSeconds int64,
	minimumBillableSeconds int64,
) (Quote, error) {
	quote := Quote{
		Currency:                "USD",
		WholesaleRateMicros:     wholesaleRateMicros,
		MarkupBasisPoints:       markupBasisPoints,
		BillingIncrementSeconds: billingIncrementSeconds,
		MinimumBillableSeconds:  minimumBillableSeconds,
	}
	if err := validatePolicy(quote); err != nil {
		return Quote{}, err
	}
	customerRate := new(big.Int).Mul(
		big.NewInt(wholesaleRateMicros),
		big.NewInt(markupBasisPoints),
	)
	customerRate = ceilDiv(customerRate, 10_000)
	customerRate.Add(customerRate, big.NewInt(wholesaleRateMicros))
	if !customerRate.IsInt64() || customerRate.Sign() <= 0 {
		return Quote{}, fmt.Errorf("customer rate exceeds supported range")
	}
	quote.CustomerRateMicros = customerRate.Int64()
	return quote, nil
}

// Validate prevents a caller from substituting an arbitrary customer rate
// or a negative margin for the quoted wholesale price and markup policy.
func (quote Quote) Validate() error {
	if err := validatePolicy(quote); err != nil {
		return err
	}
	expected, err := QuoteFromWholesale(
		quote.WholesaleRateMicros,
		quote.MarkupBasisPoints,
		quote.BillingIncrementSeconds,
		quote.MinimumBillableSeconds,
	)
	if err != nil {
		return err
	}
	if quote.CustomerRateMicros != expected.CustomerRateMicros {
		return fmt.Errorf("customer tariff does not match the wholesale markup")
	}
	return nil
}

func validatePolicy(quote Quote) error {
	if quote.Currency != "USD" || quote.WholesaleRateMicros <= 0 {
		return fmt.Errorf("pricing requires a positive USD wholesale rate")
	}
	if quote.MarkupBasisPoints <= 0 || quote.MarkupBasisPoints > 100_000 {
		return fmt.Errorf("pricing requires an approved positive markup")
	}
	if quote.BillingIncrementSeconds <= 0 || quote.BillingIncrementSeconds > 3_600 {
		return fmt.Errorf("invalid customer billing increment")
	}
	if quote.MinimumBillableSeconds < 0 || quote.MinimumBillableSeconds > 86_400 {
		return fmt.Errorf("invalid minimum billable duration")
	}
	return nil
}

// Rate charges actual usage, rounded to the tariff's customer billing
// increment and minimum. Zero usage is zero charge; the caller must still
// independently verify that the carrier did not establish a billable call.
func (quote Quote) Rate(actualSeconds int64) (Rating, error) {
	if err := quote.Validate(); err != nil {
		return Rating{}, err
	}
	if actualSeconds < 0 || actualSeconds > 86_400 {
		return Rating{}, fmt.Errorf("usage duration is out of range")
	}
	rating := Rating{
		ActualSeconds: actualSeconds,
	}
	if actualSeconds == 0 {
		return rating, nil
	}
	billableSeconds := max(actualSeconds, quote.MinimumBillableSeconds)
	increments := billableSeconds / quote.BillingIncrementSeconds
	if billableSeconds%quote.BillingIncrementSeconds != 0 {
		increments++
	}
	if increments > math.MaxInt64/quote.BillingIncrementSeconds {
		return Rating{}, fmt.Errorf("billable duration exceeds supported range")
	}
	rating.BillableSeconds = increments * quote.BillingIncrementSeconds
	customer, err := billedMinor(quote.CustomerRateMicros, rating.BillableSeconds)
	if err != nil {
		return Rating{}, err
	}
	wholesale, err := billedMinor(quote.WholesaleRateMicros, rating.BillableSeconds)
	if err != nil {
		return Rating{}, err
	}
	rating.CustomerAmountMinor = customer
	rating.EstimatedWholesaleMinor = wholesale
	return rating, nil
}

func billedMinor(rateMicros, seconds int64) (int64, error) {
	if rateMicros <= 0 || seconds < 0 {
		return 0, fmt.Errorf("invalid rate or billable duration")
	}
	if seconds == 0 {
		return 0, nil
	}
	// USD has 100 minor units; rate is in micros per minute.
	// Round once, at the end of the billable call, to a whole cent.
	total := new(big.Int).Mul(big.NewInt(rateMicros), big.NewInt(seconds))
	minor := ceilDiv(total, 600_000)
	if !minor.IsInt64() || minor.Sign() <= 0 {
		return 0, fmt.Errorf("rated amount exceeds supported range")
	}
	return minor.Int64(), nil
}

func ceilDiv(nonnegative *big.Int, divisor int64) *big.Int {
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(nonnegative, big.NewInt(divisor), remainder)
	if remainder.Sign() != 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	return quotient
}
