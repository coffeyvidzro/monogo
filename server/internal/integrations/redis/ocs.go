package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	ocsKeyPrefix   = "leamout:ocs"
	ocsEventStream = ocsKeyPrefix + ":events"

	maxOCSMicros int64 = 9_000_000_000_000_000
)

type OCS struct {
	client *redisv9.Client
}

type OCSWalletSeed struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Currency       string
	Status         string
	BalanceMicros  int64
	ReservedMicros int64
	Version        int64
}

type OCSReserveRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Currency       string
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	ChargingMode   string
	OccurredAt     time.Time
}

type OCSChargeMutationRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Currency       string
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	AmountMicros   int64
	OccurredAt     time.Time
}

type OCSFinalizeRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Currency       string
	ChargeID       uuid.UUID
	OperationID    uuid.UUID
	Status         string
	OccurredAt     time.Time
}

type OCSCreditRequest struct {
	OrganizationID uuid.UUID
	WalletID       uuid.UUID
	Currency       string
	OperationID    uuid.UUID
	AmountMicros   int64
	OccurredAt     time.Time
}

type OCSResult struct {
	Code                   string
	StreamID               string
	WalletVersion          int64
	ChargeSequence         int64
	BalanceMicros          int64
	WalletReservedMicros   int64
	ChargeAuthorizedMicros int64
	ChargeConsumedMicros   int64
	ChargeReservedMicros   int64
	ChargeStatus           string
}

func (c *Client) NewOCS() (*OCS, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	return &OCS{
		client: c.client,
	}, nil
}

func (o *OCS) EnsureWallet(
	ctx context.Context,
	seed OCSWalletSeed,
) (OCSResult, error) {
	if err := o.validateContext(ctx); err != nil {
		return OCSResult{}, err
	}
	if err := validateOCSWalletSeed(seed); err != nil {
		return OCSResult{}, err
	}

	result, err := ensureOCSWalletScript.Run(
		ctx,
		o.client,
		[]string{
			ocsWalletKey(seed.WalletID),
		},
		seed.OrganizationID.String(),
		seed.WalletID.String(),
		seed.Currency,
		seed.Status,
		seed.BalanceMicros,
		seed.ReservedMicros,
		seed.Version,
	).Result()
	if err != nil {
		return OCSResult{}, fmt.Errorf(
			"ensure OCS wallet: %w",
			err,
		)
	}

	return parseOCSResult(result)
}

func (o *OCS) Reserve(
	ctx context.Context,
	req OCSReserveRequest,
) (OCSResult, error) {
	if err := o.validateContext(ctx); err != nil {
		return OCSResult{}, err
	}
	if err := validateOCSReserveRequest(req); err != nil {
		return OCSResult{}, err
	}

	return o.runChargeMutation(
		ctx,
		reserveOCSScript,
		"reserve",
		req.OrganizationID,
		req.WalletID,
		req.Currency,
		req.ChargeID,
		req.OperationID,
		req.AmountMicros,
		req.ChargingMode,
		"",
		req.OccurredAt,
	)
}

func (o *OCS) Consume(
	ctx context.Context,
	req OCSChargeMutationRequest,
) (OCSResult, error) {
	if err := validateOCSChargeMutationRequest(req); err != nil {
		return OCSResult{}, err
	}

	return o.runChargeMutation(
		ctx,
		consumeOCSScript,
		"consume",
		req.OrganizationID,
		req.WalletID,
		req.Currency,
		req.ChargeID,
		req.OperationID,
		req.AmountMicros,
		"",
		"",
		req.OccurredAt,
	)
}

func (o *OCS) Release(
	ctx context.Context,
	req OCSChargeMutationRequest,
) (OCSResult, error) {
	if err := validateOCSChargeMutationRequest(req); err != nil {
		return OCSResult{}, err
	}

	return o.runChargeMutation(
		ctx,
		releaseOCSScript,
		"release",
		req.OrganizationID,
		req.WalletID,
		req.Currency,
		req.ChargeID,
		req.OperationID,
		req.AmountMicros,
		"",
		"",
		req.OccurredAt,
	)
}

func (o *OCS) Debit(
	ctx context.Context,
	req OCSChargeMutationRequest,
) (OCSResult, error) {
	if err := validateOCSChargeMutationRequest(req); err != nil {
		return OCSResult{}, err
	}

	return o.runChargeMutation(
		ctx,
		debitOCSScript,
		"debit",
		req.OrganizationID,
		req.WalletID,
		req.Currency,
		req.ChargeID,
		req.OperationID,
		req.AmountMicros,
		"discrete",
		"",
		req.OccurredAt,
	)
}

func (o *OCS) Finalize(
	ctx context.Context,
	req OCSFinalizeRequest,
) (OCSResult, error) {
	if err := validateOCSFinalizeRequest(req); err != nil {
		return OCSResult{}, err
	}

	return o.runChargeMutation(
		ctx,
		finalizeOCSScript,
		"finalize",
		req.OrganizationID,
		req.WalletID,
		req.Currency,
		req.ChargeID,
		req.OperationID,
		0,
		"",
		req.Status,
		req.OccurredAt,
	)
}

func (o *OCS) Credit(
	ctx context.Context,
	req OCSCreditRequest,
) (OCSResult, error) {
	if err := o.validateContext(ctx); err != nil {
		return OCSResult{}, err
	}
	if err := validateOCSCreditRequest(req); err != nil {
		return OCSResult{}, err
	}

	result, err := creditOCSScript.Run(
		ctx,
		o.client,
		[]string{
			ocsWalletKey(req.WalletID),
			ocsOperationKey(req.OperationID),
			ocsEventStream,
		},
		req.OrganizationID.String(),
		req.WalletID.String(),
		req.Currency,
		req.OperationID.String(),
		req.AmountMicros,
		req.OccurredAt.UTC().UnixMilli(),
		maxOCSMicros,
	).Result()
	if err != nil {
		return OCSResult{}, fmt.Errorf(
			"credit OCS wallet: %w",
			err,
		)
	}

	return parseOCSResult(result)
}

func (o *OCS) runChargeMutation(
	ctx context.Context,
	script *redisv9.Script,
	operation string,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	currency string,
	chargeID uuid.UUID,
	operationID uuid.UUID,
	amountMicros int64,
	chargingMode string,
	terminalStatus string,
	occurredAt time.Time,
) (OCSResult, error) {
	if err := o.validateContext(ctx); err != nil {
		return OCSResult{}, err
	}

	result, err := script.Run(
		ctx,
		o.client,
		[]string{
			ocsWalletKey(walletID),
			ocsChargeKey(chargeID),
			ocsOperationKey(operationID),
			ocsEventStream,
		},
		organizationID.String(),
		walletID.String(),
		currency,
		chargeID.String(),
		operationID.String(),
		amountMicros,
		chargingMode,
		terminalStatus,
		occurredAt.UTC().UnixMilli(),
		operation,
		maxOCSMicros,
	).Result()
	if err != nil {
		return OCSResult{}, fmt.Errorf(
			"%s OCS charge: %w",
			operation,
			err,
		)
	}

	return parseOCSResult(result)
}

func (o *OCS) validateContext(ctx context.Context) error {
	if o == nil || o.client == nil {
		return fmt.Errorf("OCS redis client is nil")
	}
	if ctx == nil {
		return fmt.Errorf("OCS context is nil")
	}

	return nil
}

func ocsWalletKey(walletID uuid.UUID) string {
	return ocsKeyPrefix + ":wallet:" + walletID.String()
}

func ocsChargeKey(chargeID uuid.UUID) string {
	return ocsKeyPrefix + ":charge:" + chargeID.String()
}

func ocsOperationKey(operationID uuid.UUID) string {
	return ocsKeyPrefix + ":operation:" + operationID.String()
}

func parseOCSResult(value any) (OCSResult, error) {
	values, ok := value.([]any)
	if !ok {
		return OCSResult{}, fmt.Errorf(
			"decode OCS result: expected array",
		)
	}
	if len(values) != 10 {
		return OCSResult{}, fmt.Errorf(
			"decode OCS result: expected 10 fields, got %d",
			len(values),
		)
	}

	walletVersion, err := ocsInt64(values[2])
	if err != nil {
		return OCSResult{}, err
	}
	chargeSequence, err := ocsInt64(values[3])
	if err != nil {
		return OCSResult{}, err
	}
	balanceMicros, err := ocsInt64(values[4])
	if err != nil {
		return OCSResult{}, err
	}
	walletReservedMicros, err := ocsInt64(values[5])
	if err != nil {
		return OCSResult{}, err
	}
	chargeAuthorizedMicros, err := ocsInt64(values[6])
	if err != nil {
		return OCSResult{}, err
	}
	chargeConsumedMicros, err := ocsInt64(values[7])
	if err != nil {
		return OCSResult{}, err
	}
	chargeReservedMicros, err := ocsInt64(values[8])
	if err != nil {
		return OCSResult{}, err
	}

	return OCSResult{
		Code:                   fmt.Sprint(values[0]),
		StreamID:               fmt.Sprint(values[1]),
		WalletVersion:          walletVersion,
		ChargeSequence:         chargeSequence,
		BalanceMicros:          balanceMicros,
		WalletReservedMicros:   walletReservedMicros,
		ChargeAuthorizedMicros: chargeAuthorizedMicros,
		ChargeConsumedMicros:   chargeConsumedMicros,
		ChargeReservedMicros:   chargeReservedMicros,
		ChargeStatus:           fmt.Sprint(values[9]),
	}, nil
}

func ocsInt64(value any) (int64, error) {
	text := fmt.Sprint(value)

	parsed, err := strconv.ParseInt(
		text,
		10,
		64,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"decode OCS integer %q: %w",
			text,
			err,
		)
	}

	return parsed, nil
}
