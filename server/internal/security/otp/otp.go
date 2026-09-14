// Package otp provides cryptographically secure one-time-code generation.
package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const maxNumericDigits = 18

// GenerateNumeric returns a zero-padded numeric code with the requested number
// of digits. crypto/rand.Int samples uniformly and avoids modulo bias.
func GenerateNumeric(digits int) (string, error) {
	if digits <= 0 || digits > maxNumericDigits {
		return "", fmt.Errorf("OTP digits must be between 1 and %d", maxNumericDigits)
	}

	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", fmt.Errorf("generate OTP: %w", err)
	}

	return fmt.Sprintf("%0*d", digits, value.Int64()), nil
}
