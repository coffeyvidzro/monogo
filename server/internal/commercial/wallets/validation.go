package wallets

import (
	"math"
	"regexp"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

var referenceLabel = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func validateEntry(entry Entry) error {
	if entry.OrganizationID == uuid.Nil || !validCurrency(entry.Currency) {
		return apperror.NewBadRequest("valid organization and currency are required")
	}
	if entry.Direction != "credit" && entry.Direction != "debit" {
		return apperror.NewBadRequest("wallet direction must be credit or debit")
	}
	if entry.AmountMinor <= 0 {
		return apperror.NewBadRequest("wallet amount must be positive")
	}
	if !referenceLabel.MatchString(entry.Reason) ||
		!referenceLabel.MatchString(entry.ReferenceType) ||
		entry.ReferenceID == uuid.Nil {
		return apperror.NewBadRequest("valid wallet reason and durable reference are required")
	}
	return nil
}

func validCurrency(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, letter := range currency {
		if letter < 'A' || letter > 'Z' {
			return false
		}
	}
	return true
}

func nextBalance(current int64, direction string, amount int64) (int64, error) {
	if current < 0 || amount <= 0 {
		return 0, apperror.NewBadRequest("invalid prepaid balance or amount")
	}
	switch direction {
	case "debit":
		if amount > current {
			return 0, apperror.NewPaymentRequired("insufficient prepaid balance")
		}
		return current - amount, nil
	case "credit":
		if amount > math.MaxInt64-current {
			return 0, apperror.NewConflict("prepaid balance limit exceeded")
		}
		return current + amount, nil
	default:
		return 0, apperror.NewBadRequest("invalid prepaid transaction direction")
	}
}
