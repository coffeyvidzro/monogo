package redis

import (
	"fmt"
	"strings"
)

func validateOCSWalletSeed(seed OCSWalletSeed) error {
	if seed.OrganizationID == [16]byte{} ||
		seed.WalletID == [16]byte{} {
		return fmt.Errorf(
			"OCS organization and wallet are required",
		)
	}
	if err := validateOCSCurrency(seed.Currency); err != nil {
		return err
	}

	switch seed.Status {
	case "active":
	case "frozen":
	case "closed":
	default:
		return fmt.Errorf(
			"OCS wallet status is invalid",
		)
	}

	if seed.BalanceMicros < 0 ||
		seed.BalanceMicros > maxOCSMicros {
		return fmt.Errorf(
			"OCS wallet balance is outside the supported range",
		)
	}
	if seed.ReservedMicros < 0 ||
		seed.ReservedMicros > seed.BalanceMicros {
		return fmt.Errorf(
			"OCS wallet reserved balance is invalid",
		)
	}
	if seed.Version < 0 {
		return fmt.Errorf(
			"OCS wallet version cannot be negative",
		)
	}

	return nil
}

func validateOCSReserveRequest(req OCSReserveRequest) error {
	if req.OrganizationID == [16]byte{} ||
		req.WalletID == [16]byte{} ||
		req.ChargeID == [16]byte{} ||
		req.OperationID == [16]byte{} {
		return fmt.Errorf(
			"OCS organization, wallet, charge, and operation are required",
		)
	}
	if err := validateOCSCurrency(req.Currency); err != nil {
		return err
	}
	if err := validateOCSAmount(req.AmountMicros); err != nil {
		return err
	}

	switch req.ChargingMode {
	case "rolling":
	case "discrete":
	default:
		return fmt.Errorf(
			"OCS charging mode is invalid",
		)
	}

	if req.OccurredAt.IsZero() {
		return fmt.Errorf(
			"OCS occurrence time is required",
		)
	}

	return nil
}

func validateOCSChargeMutationRequest(
	req OCSChargeMutationRequest,
) error {
	if req.OrganizationID == [16]byte{} ||
		req.WalletID == [16]byte{} ||
		req.ChargeID == [16]byte{} ||
		req.OperationID == [16]byte{} {
		return fmt.Errorf(
			"OCS organization, wallet, charge, and operation are required",
		)
	}
	if err := validateOCSCurrency(req.Currency); err != nil {
		return err
	}
	if err := validateOCSAmount(req.AmountMicros); err != nil {
		return err
	}
	if req.OccurredAt.IsZero() {
		return fmt.Errorf(
			"OCS occurrence time is required",
		)
	}

	return nil
}

func validateOCSFinalizeRequest(
	req OCSFinalizeRequest,
) error {
	if req.OrganizationID == [16]byte{} ||
		req.WalletID == [16]byte{} ||
		req.ChargeID == [16]byte{} ||
		req.OperationID == [16]byte{} {
		return fmt.Errorf(
			"OCS organization, wallet, charge, and operation are required",
		)
	}
	if err := validateOCSCurrency(req.Currency); err != nil {
		return err
	}

	switch req.Status {
	case "completed":
	case "failed":
	case "cancelled":
	default:
		return fmt.Errorf(
			"OCS terminal charge status is invalid",
		)
	}

	if req.OccurredAt.IsZero() {
		return fmt.Errorf(
			"OCS occurrence time is required",
		)
	}

	return nil
}

func validateOCSCreditRequest(
	req OCSCreditRequest,
) error {
	if req.OrganizationID == [16]byte{} ||
		req.WalletID == [16]byte{} ||
		req.OperationID == [16]byte{} {
		return fmt.Errorf(
			"OCS organization, wallet, and operation are required",
		)
	}
	if err := validateOCSCurrency(req.Currency); err != nil {
		return err
	}
	if err := validateOCSAmount(req.AmountMicros); err != nil {
		return err
	}
	if req.OccurredAt.IsZero() {
		return fmt.Errorf(
			"OCS occurrence time is required",
		)
	}

	return nil
}

func validateOCSCurrency(value string) error {
	value = strings.TrimSpace(value)
	if len(value) != 3 {
		return fmt.Errorf(
			"OCS currency must be a 3-letter code",
		)
	}

	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return fmt.Errorf(
				"OCS currency must be uppercase ASCII",
			)
		}
	}

	return nil
}

func validateOCSAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf(
			"OCS amount must be positive",
		)
	}
	if amount > maxOCSMicros {
		return fmt.Errorf(
			"OCS amount exceeds the supported range",
		)
	}

	return nil
}
