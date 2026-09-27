package charging

import (
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

const maxMicros int64 = 9_000_000_000_000_000

func validateReserve(req ReserveRequest) error {
	if err := validateChargeOperation(
		req.OrganizationID,
		req.ChargeID,
		req.OperationID,
		req.AmountMicros,
	); err != nil {
		return err
	}

	if req.OccurredAt.IsZero() {
		return apperror.NewBadRequest(
			"occurred_at is required",
		)
	}

	return nil
}

func validateMutation(req MutationRequest) error {
	if err := validateChargeOperation(
		req.OrganizationID,
		req.ChargeID,
		req.OperationID,
		req.AmountMicros,
	); err != nil {
		return err
	}

	if req.OccurredAt.IsZero() {
		return apperror.NewBadRequest(
			"occurred_at is required",
		)
	}

	return nil
}

func validateFinalize(req FinalizeRequest) error {
	if req.OrganizationID == uuid.Nil ||
		req.ChargeID == uuid.Nil ||
		req.OperationID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization, charge, and operation are required",
		)
	}

	switch req.Status {
	case "completed":
	case "failed":
	case "cancelled":
	default:
		return apperror.NewBadRequest(
			"final charge status must be completed, failed, or cancelled",
		)
	}

	if req.OccurredAt.IsZero() {
		return apperror.NewBadRequest(
			"occurred_at is required",
		)
	}

	return nil
}

func validateCredit(req CreditRequest) error {
	if req.OrganizationID == uuid.Nil ||
		req.WalletID == uuid.Nil ||
		req.OperationID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization, wallet, and operation are required",
		)
	}
	if err := validateAmount(req.AmountMicros); err != nil {
		return err
	}
	if req.OccurredAt.IsZero() {
		return apperror.NewBadRequest(
			"occurred_at is required",
		)
	}

	return nil
}

func validateChargeOperation(
	organizationID uuid.UUID,
	chargeID uuid.UUID,
	operationID uuid.UUID,
	amountMicros int64,
) error {
	if organizationID == uuid.Nil ||
		chargeID == uuid.Nil ||
		operationID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization, charge, and operation are required",
		)
	}

	return validateAmount(amountMicros)
}

func validateAmount(amountMicros int64) error {
	if amountMicros <= 0 {
		return apperror.NewBadRequest(
			"amount_micros must be positive",
		)
	}
	if amountMicros > maxMicros {
		return apperror.NewBadRequest(
			"amount_micros exceeds the supported range",
		)
	}

	return nil
}
