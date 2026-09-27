package charging

import (
	"time"

	"github.com/google/uuid"
)

type ReserveRequest struct {
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	OccurredAt     time.Time
}

type MutationRequest struct {
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	OccurredAt     time.Time
}

type FinalizeRequest struct {
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	Status         string
	OccurredAt     time.Time
}

type CreditRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	OccurredAt     time.Time
}

type Result struct {
	Code                 string
	StreamID             string
	WalletVersion        int64
	ChargeSequence       int64
	BalanceMicros        int64
	WalletReservedMicros int64
	AuthorizedMicros     int64
	ConsumedMicros       int64
	ChargeReservedMicros int64
	ChargeStatus         string
}

// OperationRequest is the HTTP representation shared by reserve, consume,
// release, and debit operations. OperationID is the idempotency key for an
// individual balance mutation.
type OperationRequest struct {
	OperationID  uuid.UUID `json:"operation_id"`
	AmountMicros int64     `json:"amount_micros"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type FinalizeOperationRequest struct {
	OperationID uuid.UUID `json:"operation_id"`
	Status      string    `json:"status"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type CreditOperationRequest struct {
	OperationID  uuid.UUID `json:"operation_id"`
	AmountMicros int64     `json:"amount_micros"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type Response struct {
	Code                 string `json:"code"`
	StreamID             string `json:"stream_id"`
	WalletVersion        int64  `json:"wallet_version"`
	ChargeSequence       int64  `json:"charge_sequence,omitempty"`
	BalanceMicros        int64  `json:"balance_micros"`
	WalletReservedMicros int64  `json:"wallet_reserved_micros"`
	AuthorizedMicros     int64  `json:"authorized_micros,omitempty"`
	ConsumedMicros       int64  `json:"consumed_micros,omitempty"`
	ChargeReservedMicros int64  `json:"charge_reserved_micros,omitempty"`
	ChargeStatus         string `json:"charge_status,omitempty"`
}

func response(value Result) Response {
	return Response{
		Code:                 value.Code,
		StreamID:             value.StreamID,
		WalletVersion:        value.WalletVersion,
		ChargeSequence:       value.ChargeSequence,
		BalanceMicros:        value.BalanceMicros,
		WalletReservedMicros: value.WalletReservedMicros,
		AuthorizedMicros:     value.AuthorizedMicros,
		ConsumedMicros:       value.ConsumedMicros,
		ChargeReservedMicros: value.ChargeReservedMicros,
		ChargeStatus:         value.ChargeStatus,
	}
}
