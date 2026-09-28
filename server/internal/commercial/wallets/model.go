package wallets

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive = "active"
	StatusFrozen = "frozen"
	StatusClosed = "closed"

	DirectionCredit = "credit"
	DirectionDebit  = "debit"
)

var (
	ErrNotFound            = errors.New("wallet not found")
	ErrInvalidState        = errors.New("wallet state does not allow operation")
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrOperationConflict   = errors.New("wallet operation conflicts with existing ledger entry")
)

type Wallet struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Currency       string
	Status         string
	BalanceMicros  int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LedgerEntry struct {
	ID                 uuid.UUID
	WalletID           uuid.UUID
	OrganizationID     uuid.UUID
	OperationID        uuid.UUID
	Direction          string
	Reason             string
	AmountMicros       int64
	BalanceAfterMicros int64
	ReferenceType      *string
	ReferenceID        *uuid.UUID
	OccurredAt         time.Time
	CreatedAt          time.Time
}

type CreateRequest struct {
	OrganizationID uuid.UUID
	Currency       string
}

type MovementRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	Reason         string
	ReferenceType  *string
	ReferenceID    *uuid.UUID
	OccurredAt     time.Time
}

type ListLedgerRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Limit          int32
	Offset         int32
}
