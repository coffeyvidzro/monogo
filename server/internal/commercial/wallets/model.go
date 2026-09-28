package wallets

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	CurrencyUSD = "USD"

	StatusActive = "active"
	StatusFrozen = "frozen"
	StatusClosed = "closed"

	DirectionCredit = "credit"
	DirectionDebit  = "debit"

	HoldStatusActive   = "active"
	HoldStatusCaptured = "captured"
	HoldStatusReleased = "released"
)

var (
	ErrInvalidInput        = errors.New("invalid wallet input")
	ErrNotFound            = errors.New("wallet not found")
	ErrInvalidState        = errors.New("wallet state does not allow operation")
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrOperationConflict   = errors.New("wallet operation conflicts with existing ledger entry")
)

type Wallet struct {
	ID              uuid.UUID `json:"id"`
	OrganizationID  uuid.UUID `json:"organization_id"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	BalanceMicros   int64     `json:"balance_micros"`
	ReservedMicros  int64     `json:"reserved_micros"`
	AvailableMicros int64     `json:"available_micros"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Hold struct {
	ID             uuid.UUID
	WalletID       uuid.UUID
	OrganizationID uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	Reason         string
	ReferenceType  *string
	ReferenceID    *uuid.UUID
	Status         string
	CapturedAt     *time.Time
	ReleasedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LedgerEntry struct {
	ID                 uuid.UUID  `json:"id"`
	WalletID           uuid.UUID  `json:"wallet_id"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	OperationID        uuid.UUID  `json:"operation_id"`
	Direction          string     `json:"direction"`
	Reason             string     `json:"reason"`
	AmountMicros       int64      `json:"amount_micros"`
	BalanceAfterMicros int64      `json:"balance_after_micros"`
	ReferenceType      *string    `json:"reference_type,omitempty"`
	ReferenceID        *uuid.UUID `json:"reference_id,omitempty"`
	OccurredAt         time.Time  `json:"occurred_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

type CreateRequest struct {
	OrganizationID uuid.UUID
}

type MovementRequest struct {
	OrganizationID uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	Reason         string
	ReferenceType  *string
	ReferenceID    *uuid.UUID
	OccurredAt     time.Time
}

type HoldRequest struct {
	OrganizationID uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	Reason         string
	ReferenceType  *string
	ReferenceID    *uuid.UUID
}

type ListLedgerRequest struct {
	OrganizationID uuid.UUID
	Limit          int32
	Offset         int32
}
