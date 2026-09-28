package wallets

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var ledgerTokenPattern = regexp.MustCompile("^[a-z][a-z0-9_]{0,63}$")

func validateCreateRequest(req CreateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}

	return nil
}

func normalizeMovementRequest(req *MovementRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.OperationID == uuid.Nil {
		return fmt.Errorf("%w: operation id is required", ErrInvalidInput)
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}
	if err := normalizeOperationMetadata(
		&req.Reason,
		&req.ReferenceType,
		&req.ReferenceID,
	); err != nil {
		return err
	}

	return nil
}

func normalizeHoldRequest(req *HoldRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.OperationID == uuid.Nil {
		return fmt.Errorf("%w: operation id is required", ErrInvalidInput)
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
	}
	if err := normalizeOperationMetadata(
		&req.Reason,
		&req.ReferenceType,
		&req.ReferenceID,
	); err != nil {
		return err
	}

	return nil
}

func validateHoldOperation(
	organizationID uuid.UUID,
	operationID uuid.UUID,
) error {
	if organizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if operationID == uuid.Nil {
		return fmt.Errorf("%w: operation id is required", ErrInvalidInput)
	}

	return nil
}

func normalizeOperationMetadata(
	reason *string,
	referenceType **string,
	referenceID **uuid.UUID,
) error {
	*reason = strings.ToLower(strings.TrimSpace(*reason))
	if !ledgerTokenPattern.MatchString(*reason) {
		return fmt.Errorf("%w: reason is invalid", ErrInvalidInput)
	}

	if (*referenceType == nil) != (*referenceID == nil) {
		return fmt.Errorf("%w: reference type and reference id must be provided together", ErrInvalidInput)
	}
	if *referenceType != nil {
		value := strings.ToLower(strings.TrimSpace(**referenceType))
		if !ledgerTokenPattern.MatchString(value) {
			return fmt.Errorf("%w: reference type is invalid", ErrInvalidInput)
		}
		if **referenceID == uuid.Nil {
			return fmt.Errorf("%w: reference id is required", ErrInvalidInput)
		}
		*referenceType = &value
	}

	return nil
}

func normalizeListLedgerRequest(req *ListLedgerRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}
	if req.Offset < 0 {
		return fmt.Errorf("%w: offset cannot be negative", ErrInvalidInput)
	}
	if req.Limit == 0 {
		req.Limit = 50
	}
	if req.Limit < 1 || req.Limit > 100 {
		return fmt.Errorf("%w: limit must be between 1 and 100", ErrInvalidInput)
	}

	return nil
}
