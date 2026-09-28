package wallets

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var ledgerTokenPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func normalizeCreateRequest(req *CreateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("currency must be a three-letter code")
	}

	return nil
}

func normalizeMovementRequest(req *MovementRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if req.WalletID == uuid.Nil {
		return fmt.Errorf("wallet id is required")
	}
	if req.OperationID == uuid.Nil {
		return fmt.Errorf("operation id is required")
	}
	if req.AmountMicros <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	req.Reason = strings.ToLower(strings.TrimSpace(req.Reason))
	if !ledgerTokenPattern.MatchString(req.Reason) {
		return fmt.Errorf("reason is invalid")
	}

	if (req.ReferenceType == nil) != (req.ReferenceID == nil) {
		return fmt.Errorf("reference type and reference id must be provided together")
	}
	if req.ReferenceType != nil {
		value := strings.ToLower(strings.TrimSpace(*req.ReferenceType))
		if !ledgerTokenPattern.MatchString(value) {
			return fmt.Errorf("reference type is invalid")
		}
		if *req.ReferenceID == uuid.Nil {
			return fmt.Errorf("reference id is required")
		}
		req.ReferenceType = &value
	}

	return nil
}

func normalizeListLedgerRequest(req *ListLedgerRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if req.WalletID == uuid.Nil {
		return fmt.Errorf("wallet id is required")
	}
	if req.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	if req.Limit == 0 {
		req.Limit = 50
	}
	if req.Limit < 1 || req.Limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}

	return nil
}
