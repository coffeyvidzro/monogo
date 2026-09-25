package wallets

import "github.com/google/uuid"

// Entry describes one successful balance-changing operation. ReferenceID must
// be stable across retries of the same business operation.
type Entry struct {
	OrganizationID uuid.UUID
	Currency       string
	Direction      string
	Reason         string
	AmountMinor    int64
	ReferenceType  string
	ReferenceID    uuid.UUID
}
