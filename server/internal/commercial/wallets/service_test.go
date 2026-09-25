package wallets

import (
	"errors"
	"math"
	"testing"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func TestNextBalance(t *testing.T) {
	tests := []struct {
		name      string
		current   int64
		direction string
		amount    int64
		want      int64
		code      string
	}{
		{name: "credit", current: 100, direction: "credit", amount: 25, want: 125},
		{name: "debit", current: 100, direction: "debit", amount: 25, want: 75},
		{name: "empty", current: 25, direction: "debit", amount: 25, want: 0},
		{name: "insufficient", current: 10, direction: "debit", amount: 11, code: "PAYMENT_REQUIRED"},
		{name: "overflow", current: math.MaxInt64, direction: "credit", amount: 1, code: "CONFLICT"},
		{name: "negative balance", current: -1, direction: "debit", amount: 1, code: "BAD_REQUEST"},
		{name: "zero amount", current: 10, direction: "credit", amount: 0, code: "BAD_REQUEST"},
		{name: "unknown direction", current: 10, direction: "adjustment", amount: 1, code: "BAD_REQUEST"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := nextBalance(tc.current, tc.direction, tc.amount)
			if tc.code == "" {
				if err != nil || got != tc.want {
					t.Fatalf("nextBalance() = %d, %v; want %d", got, err, tc.want)
				}
				return
			}
			var appErr *apperror.AppError
			if !errors.As(err, &appErr) || appErr.Code != tc.code {
				t.Fatalf("nextBalance() error = %v; want %s", err, tc.code)
			}
		})
	}
}

func TestValidateEntry(t *testing.T) {
	valid := Entry{
		OrganizationID: uuid.New(),
		Currency:       "USD",
		Direction:      "debit",
		Reason:         "number_purchase",
		AmountMinor:    100,
		ReferenceType:  "managed_number_order",
		ReferenceID:    uuid.New(),
	}
	if err := validateEntry(valid); err != nil {
		t.Fatal(err)
	}

	bad := []Entry{
		{Currency: "USD", Direction: "credit", Reason: "topup", AmountMinor: 100, ReferenceType: "payment", ReferenceID: uuid.New()},
		{OrganizationID: valid.OrganizationID, Currency: "US", Direction: "credit", Reason: "topup", AmountMinor: 100, ReferenceType: "payment", ReferenceID: uuid.New()},
		{OrganizationID: valid.OrganizationID, Currency: "usd", Direction: "credit", Reason: "topup", AmountMinor: 100, ReferenceType: "payment", ReferenceID: uuid.New()},
		{OrganizationID: valid.OrganizationID, Currency: "USD", Direction: "credit", Reason: "topup", AmountMinor: 0, ReferenceType: "payment", ReferenceID: uuid.New()},
		{OrganizationID: valid.OrganizationID, Currency: "USD", Direction: "credit", Reason: "topup", AmountMinor: 100, ReferenceType: "payment"},
		{OrganizationID: valid.OrganizationID, Currency: "USD", Direction: "credit", Reason: "topup", AmountMinor: 100, ReferenceType: "Bad Ref", ReferenceID: uuid.New()},
	}
	for _, entry := range bad {
		if err := validateEntry(entry); err == nil {
			t.Fatalf("invalid entry accepted: %+v", entry)
		}
	}
}
