package ledger

import (
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func validateEntryIdentity(
	organizationID uuid.UUID,
	entryID uuid.UUID,
) error {
	if organizationID == uuid.Nil ||
		entryID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and ledger entry are required",
		)
	}

	return nil
}

func validateWalletIdentity(
	organizationID uuid.UUID,
	walletID uuid.UUID,
) error {
	if organizationID == uuid.Nil ||
		walletID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and wallet are required",
		)
	}

	return nil
}

func validateChargeIdentity(
	organizationID uuid.UUID,
	chargeID uuid.UUID,
) error {
	if organizationID == uuid.Nil ||
		chargeID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and charge are required",
		)
	}

	return nil
}

func normalizeListLimit(limit int32) (int32, error) {
	if limit == 0 {
		return defaultListLimit, nil
	}

	if limit < 1 || limit > maxListLimit {
		return 0, apperror.NewBadRequest(
			"ledger limit must be between 1 and 200",
		)
	}

	return limit, nil
}
