package charging

import (
	"github.com/google/uuid"
)

type ReserveRequest struct {
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
}

type MutationRequest struct {
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
}

type FinalizeRequest struct {
	OrganizationID uuid.UUID
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	Status         string
}

type CreditRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
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
