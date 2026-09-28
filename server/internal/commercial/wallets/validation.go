package wallets

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var ledgerTokenPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

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

	req.Reason = strings.ToLower(strings.TrimSpace(req.Reason))
	if !ledgerTokenPattern.MatchString(req.Reason) {
		return fmt.Errorf("%w: reason is invalid", ErrInvalidInput)
	}

	if (req.ReferenceType == nil) != (req.ReferenceID == nil) {
		return fmt.Errorf("%w: reference type and reference id must be provided together", ErrInvalidInput)
	}
	if req.ReferenceType != nil {
		value := strings.ToLower(strings.TrimSpace(*req.ReferenceType))
		if !ledgerTokenPattern.MatchString(value) {
			return fmt.Errorf("%w: reference type is invalid", ErrInvalidInput)
		}
		if *req.ReferenceID == uuid.Nil {
			return fmt.Errorf("%w: reference id is required", ErrInvalidInput)
		}
		req.ReferenceType = &value
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
