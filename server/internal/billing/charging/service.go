package charging

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/billing/charges"
	"github.com/coffeyvidzro/monogo/internal/billing/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type Service struct {
	wallets *wallets.Service
	charges *charges.Service
	ocs     *redisintegration.OCS
}

func NewService(
	walletService *wallets.Service,
	chargeService *charges.Service,
	ocs *redisintegration.OCS,
) *Service {
	if walletService == nil {
		panic("billing charging: wallet service is required")
	}
	if chargeService == nil {
		panic("billing charging: charge service is required")
	}
	if ocs == nil {
		panic("billing charging: OCS is required")
	}

	return &Service{
		wallets: walletService,
		charges: chargeService,
		ocs:     ocs,
	}
}

func (s *Service) Reserve(
	ctx context.Context,
	req ReserveRequest,
) (Result, error) {
	if err := validateReserve(req); err != nil {
		return Result{}, err
	}

	charge, wallet, err := s.loadChargeWallet(
		ctx,
		req.OrganizationID,
		req.ChargeID,
	)
	if err != nil {
		return Result{}, err
	}
	if err := s.ensureWallet(
		ctx,
		wallet,
	); err != nil {
		return Result{}, err
	}

	result, err := s.ocs.Reserve(
		ctx,
		redisintegration.OCSReserveRequest{
			OrganizationID: req.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			ChargeID:       charge.ID,
			OperationID:    req.OperationID,
			AmountMicros:   req.AmountMicros,
			ChargingMode:   charge.ChargingMode,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return Result{}, apperror.NewInternal(
			"reserve prepaid funds",
			err,
		)
	}

	return mapOCSResult(result)
}

func (s *Service) Consume(
	ctx context.Context,
	req MutationRequest,
) (Result, error) {
	if err := validateMutation(req); err != nil {
		return Result{}, err
	}

	charge, wallet, err := s.loadChargeWallet(
		ctx,
		req.OrganizationID,
		req.ChargeID,
	)
	if err != nil {
		return Result{}, err
	}
	if err := s.ensureWallet(
		ctx,
		wallet,
	); err != nil {
		return Result{}, err
	}

	result, err := s.ocs.Consume(
		ctx,
		redisintegration.OCSChargeMutationRequest{
			OrganizationID: req.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			ChargeID:       charge.ID,
			OperationID:    req.OperationID,
			AmountMicros:   req.AmountMicros,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return Result{}, apperror.NewInternal(
			"consume prepaid funds",
			err,
		)
	}

	return mapOCSResult(result)
}

func (s *Service) Release(
	ctx context.Context,
	req MutationRequest,
) (Result, error) {
	if err := validateMutation(req); err != nil {
		return Result{}, err
	}

	charge, wallet, err := s.loadChargeWallet(
		ctx,
		req.OrganizationID,
		req.ChargeID,
	)
	if err != nil {
		return Result{}, err
	}
	if err := s.ensureWallet(
		ctx,
		wallet,
	); err != nil {
		return Result{}, err
	}

	result, err := s.ocs.Release(
		ctx,
		redisintegration.OCSChargeMutationRequest{
			OrganizationID: req.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			ChargeID:       charge.ID,
			OperationID:    req.OperationID,
			AmountMicros:   req.AmountMicros,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return Result{}, apperror.NewInternal(
			"release prepaid funds",
			err,
		)
	}

	return mapOCSResult(result)
}

func (s *Service) Debit(
	ctx context.Context,
	req MutationRequest,
) (Result, error) {
	if err := validateMutation(req); err != nil {
		return Result{}, err
	}

	charge, wallet, err := s.loadChargeWallet(
		ctx,
		req.OrganizationID,
		req.ChargeID,
	)
	if err != nil {
		return Result{}, err
	}
	if charge.ChargingMode != charges.ModeDiscrete {
		return Result{}, apperror.NewConflict(
			"direct debit requires a discrete charge",
		)
	}
	if err := s.ensureWallet(
		ctx,
		wallet,
	); err != nil {
		return Result{}, err
	}

	result, err := s.ocs.Debit(
		ctx,
		redisintegration.OCSChargeMutationRequest{
			OrganizationID: req.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			ChargeID:       charge.ID,
			OperationID:    req.OperationID,
			AmountMicros:   req.AmountMicros,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return Result{}, apperror.NewInternal(
			"debit prepaid funds",
			err,
		)
	}

	return mapOCSResult(result)
}

func (s *Service) Finalize(
	ctx context.Context,
	req FinalizeRequest,
) (Result, error) {
	if err := validateFinalize(req); err != nil {
		return Result{}, err
	}

	charge, wallet, err := s.loadChargeWallet(
		ctx,
		req.OrganizationID,
		req.ChargeID,
	)
	if err != nil {
		return Result{}, err
	}
	if err := s.ensureWallet(
		ctx,
		wallet,
	); err != nil {
		return Result{}, err
	}

	result, err := s.ocs.Finalize(
		ctx,
		redisintegration.OCSFinalizeRequest{
			OrganizationID: req.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			ChargeID:       charge.ID,
			OperationID:    req.OperationID,
			Status:         req.Status,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return Result{}, apperror.NewInternal(
			"finalize prepaid charge",
			err,
		)
	}

	return mapOCSResult(result)
}

func (s *Service) Credit(
	ctx context.Context,
	req CreditRequest,
) (Result, error) {
	if err := validateCredit(req); err != nil {
		return Result{}, err
	}

	wallet, err := s.wallets.GetByID(
		ctx,
		req.OrganizationID,
		req.WalletID,
	)
	if err != nil {
		return Result{}, err
	}
	if err := s.ensureWallet(
		ctx,
		wallet,
	); err != nil {
		return Result{}, err
	}

	result, err := s.ocs.Credit(
		ctx,
		redisintegration.OCSCreditRequest{
			OrganizationID: req.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			OperationID:    req.OperationID,
			AmountMicros:   req.AmountMicros,
			OccurredAt:     req.OccurredAt,
		},
	)
	if err != nil {
		return Result{}, apperror.NewInternal(
			"credit prepaid wallet",
			err,
		)
	}

	return mapOCSResult(result)
}

func (s *Service) loadChargeWallet(
	ctx context.Context,
	organizationID uuid.UUID,
	chargeID uuid.UUID,
) (
	sqlc.Charge,
	sqlc.Wallet,
	error,
) {
	charge, err := s.charges.Get(
		ctx,
		organizationID,
		chargeID,
	)
	if err != nil {
		return sqlc.Charge{}, sqlc.Wallet{}, err
	}
	if charge.Status != "pending" &&
		charge.Status != "active" {
		return sqlc.Charge{}, sqlc.Wallet{}, apperror.NewConflict(
			"charge is already terminal",
		)
	}

	wallet, err := s.wallets.GetByID(
		ctx,
		organizationID,
		charge.WalletID,
	)
	if err != nil {
		return sqlc.Charge{}, sqlc.Wallet{}, err
	}
	if wallet.Currency != charge.Currency {
		return sqlc.Charge{}, sqlc.Wallet{}, apperror.NewInternal(
			"charge and wallet currency mismatch",
			nil,
		)
	}

	return charge, wallet, nil
}

func (s *Service) ensureWallet(
	ctx context.Context,
	wallet sqlc.Wallet,
) error {
	result, err := s.ocs.EnsureWallet(
		ctx,
		redisintegration.OCSWalletSeed{
			OrganizationID: wallet.OrganizationID,
			WalletID:       wallet.ID,
			Currency:       wallet.Currency,
			Status:         wallet.Status,
			BalanceMicros:  wallet.BalanceMicros,
			ReservedMicros: wallet.ReservedMicros,
			Version:        wallet.OcsVersion,
		},
	)
	if err != nil {
		return apperror.NewInternal(
			"initialize prepaid wallet OCS state",
			err,
		)
	}

	switch result.Code {
	case "ok":
		return nil
	case "exists":
		return nil
	case "wallet_mismatch":
		return apperror.NewInternal(
			"prepaid wallet OCS identity mismatch",
			nil,
		)
	default:
		return apperror.NewServiceUnavailable(
			"prepaid wallet OCS state is unavailable",
			fmt.Errorf(
				"unexpected OCS wallet initialization result %q",
				result.Code,
			),
		)
	}
}

func mapOCSResult(
	value redisintegration.OCSResult,
) (Result, error) {
	result := Result{
		Code:                 value.Code,
		StreamID:             value.StreamID,
		WalletVersion:        value.WalletVersion,
		ChargeSequence:       value.ChargeSequence,
		BalanceMicros:        value.BalanceMicros,
		WalletReservedMicros: value.WalletReservedMicros,
		AuthorizedMicros:     value.ChargeAuthorizedMicros,
		ConsumedMicros:       value.ChargeConsumedMicros,
		ChargeReservedMicros: value.ChargeReservedMicros,
		ChargeStatus:         value.ChargeStatus,
	}

	switch value.Code {
	case "ok":
		return result, nil
	case "replay":
		return result, nil
	case "insufficient_funds":
		return Result{}, apperror.NewPaymentRequired(
			"insufficient available prepaid balance",
		)
	case "wallet_not_active":
		return Result{}, apperror.NewConflict(
			"prepaid wallet is not active",
		)
	case "charge_not_found":
		return Result{}, apperror.NewConflict(
			"charge has no active OCS state",
		)
	case "charge_not_active":
		return Result{}, apperror.NewConflict(
			"charge is not active",
		)
	case "charge_mode_mismatch":
		return Result{}, apperror.NewConflict(
			"charge mode does not match the OCS state",
		)
	case "insufficient_reservation":
		return Result{}, apperror.NewConflict(
			"charge reservation is insufficient",
		)
	case "operation_conflict":
		return Result{}, apperror.NewConflict(
			"operation id was already used with different terms",
		)
	case "wallet_not_found":
		return Result{}, apperror.NewServiceUnavailable(
			"prepaid wallet OCS state is unavailable",
			nil,
		)
	case "wallet_mismatch":
		return Result{}, apperror.NewInternal(
			"prepaid wallet OCS identity mismatch",
			nil,
		)
	case "charge_mismatch":
		return Result{}, apperror.NewInternal(
			"prepaid charge OCS identity mismatch",
			nil,
		)
	case "overflow":
		return Result{}, apperror.NewInternal(
			"prepaid OCS amount overflow",
			nil,
		)
	default:
		return Result{}, apperror.NewInternal(
			"unexpected prepaid OCS result",
			fmt.Errorf(
				"unsupported OCS result code %q",
				value.Code,
			),
		)
	}
}
