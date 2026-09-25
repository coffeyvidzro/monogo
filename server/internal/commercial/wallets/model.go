package wallets

import (
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

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

type ReserveRequest struct {
	OrganizationID uuid.UUID
	Currency       string
	AmountMinor    int64
	OperationType  string
	OperationID    string
	ExpiresAt      time.Time
}

type ExtendRequest struct {
	OrganizationID uuid.UUID
	ReservationID  uuid.UUID
	AmountMinor    int64
	ExpiresAt      time.Time
}

type CaptureRequest struct {
	OrganizationID uuid.UUID
	ReservationID  uuid.UUID
	AmountMinor    int64
	Reason         string
	ReferenceType  string
	ReferenceID    uuid.UUID
}

type ReservationResult struct {
	Reservation sqlc.WalletReservation
	Transaction *sqlc.WalletTransaction
}

type Response struct {
	ID             uuid.UUID `json:"id"`
	Currency       string    `json:"currency"`
	BalanceMinor   int64     `json:"balance_minor"`
	ReservedMinor  int64     `json:"reserved_minor"`
	AvailableMinor int64     `json:"available_minor"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TransactionResponse struct {
	ID                uuid.UUID `json:"id"`
	WalletID          uuid.UUID `json:"wallet_id"`
	Direction         string    `json:"direction"`
	Reason            string    `json:"reason"`
	AmountMinor       int64     `json:"amount_minor"`
	BalanceAfterMinor int64     `json:"balance_after_minor"`
	ReferenceType     string    `json:"reference_type"`
	ReferenceID       uuid.UUID `json:"reference_id"`
	CreatedAt         time.Time `json:"created_at"`
}

func response(row sqlc.Wallet) Response {
	return Response{ID: row.ID, Currency: row.Currency, BalanceMinor: row.BalanceMinor,
		ReservedMinor: row.ReservedMinor, AvailableMinor: row.BalanceMinor - row.ReservedMinor,
		CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt)}
}

func transactionResponse(row sqlc.WalletTransaction) TransactionResponse {
	return TransactionResponse{ID: row.ID, WalletID: row.WalletID, Direction: row.Direction,
		Reason: row.Reason, AmountMinor: row.AmountMinor, BalanceAfterMinor: row.BalanceAfterMinor,
		ReferenceType: row.ReferenceType, ReferenceID: row.ReferenceID,
		CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt)}
}
