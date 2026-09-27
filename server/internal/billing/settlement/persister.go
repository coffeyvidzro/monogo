package settlement

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Persister struct {
	repository *Repository
}

func NewPersister(repository *Repository) *Persister {
	if repository == nil {
		panic("billing settlement: repository is required")
	}

	return &Persister{
		repository: repository,
	}
}

func (p *Persister) Persist(
	ctx context.Context,
	event redisintegration.OCSEvent,
) (Result, error) {
	if err := redisintegration.ValidateOCSEvent(event); err != nil {
		return Result{
				Outcome: OutcomeIntegrity,
			}, &PersistenceError{
				Outcome: OutcomeIntegrity,
				Cause:   fmt.Errorf("validate OCS event: %w", err),
			}
	}

	tx, queries, err := p.repository.Begin(ctx)
	if err != nil {
		return retryResult(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	walletParams := sqlc.LockOCSWalletParams{
		WalletID:       event.WalletID,
		OrganizationID: event.OrganizationID,
	}
	wallet, err := queries.LockOCSWallet(
		ctx,
		walletParams,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return integrityResult(fmt.Errorf("OCS wallet %s does not exist", event.WalletID))
	}
	if err != nil {
		return retryResult(fmt.Errorf("lock OCS wallet projection: %w", err))
	}
	if err := queries.LockOCSOperation(
		ctx,
		event.OperationID.String(),
	); err != nil {
		return retryResult(fmt.Errorf("lock OCS operation: %w", err))
	}

	existing, err := queries.GetWalletEventByOperationID(
		ctx,
		event.OperationID,
	)
	if err == nil {
		if !samePersistedEvent(existing, event) {
			return integrityResult(fmt.Errorf(
				"OCS operation %s conflicts with its persisted event",
				event.OperationID,
			))
		}

		if err := tx.Commit(ctx); err != nil {
			return retryResult(fmt.Errorf("commit OCS replay transaction: %w", err))
		}

		return Result{
			Outcome: OutcomeAlreadyApplied,
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return retryResult(fmt.Errorf("read persisted OCS operation: %w", err))
	}

	if err := validateWalletProjection(wallet, event); err != nil {
		return classifyValidationError(err)
	}

	var charge sqlc.Charge
	if event.ChargeID != nil {
		chargeParams := sqlc.LockOCSChargeParams{
			ChargeID:       *event.ChargeID,
			OrganizationID: event.OrganizationID,
			WalletID:       event.WalletID,
		}
		charge, err = queries.LockOCSCharge(
			ctx,
			chargeParams,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return integrityResult(fmt.Errorf("OCS charge %s does not exist", *event.ChargeID))
		}
		if err != nil {
			return retryResult(fmt.Errorf("lock OCS charge projection: %w", err))
		}
		if err := validateChargeProjection(wallet, charge, event); err != nil {
			return classifyValidationError(err)
		}
		if err := validateChargeTransition(charge, event); err != nil {
			return integrityResult(err)
		}
	}

	walletEvent, err := createWalletEvent(ctx, queries, event)
	if err != nil {
		return retryResult(fmt.Errorf("insert OCS wallet event: %w", err))
	}

	walletProjectionParams := sqlc.ApplyOCSWalletProjectionParams{
		BalanceAfterMicros:    event.BalanceAfterMicros,
		ReservedAfterMicros:   event.ReservedAfterMicros,
		WalletVersion:         event.WalletVersion,
		WalletID:              event.WalletID,
		OrganizationID:        event.OrganizationID,
		PreviousWalletVersion: wallet.OcsVersion,
	}
	if _, err := queries.ApplyOCSWalletProjection(
		ctx,
		walletProjectionParams,
	); err != nil {
		return retryResult(fmt.Errorf("apply OCS wallet projection: %w", err))
	}

	if event.ChargeID != nil {
		chargeProjectionParams := sqlc.ApplyOCSChargeProjectionParams{
			ChargeStatus:           event.ChargeStatus,
			AuthorizedMicros:       event.ChargeAuthorizedMicros,
			ConsumedMicros:         event.ChargeConsumedMicros,
			ReservedMicros:         event.ChargeReservedMicros,
			ChargeSequence:         event.ChargeSequence,
			ClosedAt:               terminalTime(event),
			ChargeID:               *event.ChargeID,
			OrganizationID:         event.OrganizationID,
			WalletID:               event.WalletID,
			PreviousChargeSequence: charge.OcsSequence,
		}
		if _, err := queries.ApplyOCSChargeProjection(
			ctx,
			chargeProjectionParams,
		); err != nil {
			return retryResult(fmt.Errorf("apply OCS charge projection: %w", err))
		}
	}

	if err := createLedgerEntry(ctx, queries, walletEvent.ID, event); err != nil {
		return retryResult(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return retryResult(fmt.Errorf("commit OCS event transaction: %w", err))
	}

	return Result{
		Outcome: OutcomeApplied,
	}, nil
}

func createWalletEvent(
	ctx context.Context,
	queries *sqlc.Queries,
	event redisintegration.OCSEvent,
) (sqlc.WalletEvent, error) {
	var chargeSequence *int64
	var chargeAuthorized *int64
	var chargeConsumed *int64
	var chargeReserved *int64
	var chargeStatus *string
	if event.ChargeID != nil {
		chargeSequence = &event.ChargeSequence
		chargeAuthorized = &event.ChargeAuthorizedMicros
		chargeConsumed = &event.ChargeConsumedMicros
		chargeReserved = &event.ChargeReservedMicros
		chargeStatus = &event.ChargeStatus
	}

	params := sqlc.CreateOCSWalletEventParams{
		WalletID:                    event.WalletID,
		OrganizationID:              event.OrganizationID,
		ChargeID:                    event.ChargeID,
		OperationID:                 event.OperationID,
		WalletVersion:               event.WalletVersion,
		ChargeSequence:              chargeSequence,
		EventType:                   event.EventType,
		BalanceDeltaMicros:          event.BalanceDeltaMicros,
		ReservedDeltaMicros:         event.ReservedDeltaMicros,
		BalanceAfterMicros:          event.BalanceAfterMicros,
		ReservedAfterMicros:         event.ReservedAfterMicros,
		ChargeAuthorizedAfterMicros: chargeAuthorized,
		ChargeConsumedAfterMicros:   chargeConsumed,
		ChargeReservedAfterMicros:   chargeReserved,
		ChargeStatus:                chargeStatus,
		OccurredAt:                  pgconv.TimeToTimestamptz(event.OccurredAt),
	}

	return queries.CreateOCSWalletEvent(
		ctx,
		params,
	)
}

func createLedgerEntry(
	ctx context.Context,
	queries *sqlc.Queries,
	walletEventID uuid.UUID,
	event redisintegration.OCSEvent,
) error {
	var direction string
	switch event.EventType {
	case "credit":
		direction = "credit"
	case "consume", "debit":
		direction = "debit"
	default:
		return nil
	}

	amountMicros := event.BalanceDeltaMicros
	if amountMicros < 0 {
		amountMicros = -amountMicros
	}
	if amountMicros <= 0 {
		return fmt.Errorf("settled OCS event must have a non-zero balance delta")
	}

	params := sqlc.CreateOCSLedgerEntryParams{
		WalletEventID:      walletEventID,
		WalletID:           event.WalletID,
		OrganizationID:     event.OrganizationID,
		ChargeID:           event.ChargeID,
		Direction:          direction,
		Reason:             event.EventType,
		AmountMicros:       amountMicros,
		BalanceAfterMicros: event.BalanceAfterMicros,
		OccurredAt:         pgconv.TimeToTimestamptz(event.OccurredAt),
	}
	if _, err := queries.CreateOCSLedgerEntry(
		ctx,
		params,
	); err != nil {
		return fmt.Errorf("insert OCS ledger entry: %w", err)
	}

	return nil
}

func samePersistedEvent(
	existing sqlc.WalletEvent,
	event redisintegration.OCSEvent,
) bool {
	return existing.OperationID == event.OperationID &&
		existing.WalletID == event.WalletID &&
		existing.OrganizationID == event.OrganizationID &&
		equalOptionalUUID(existing.ChargeID, event.ChargeID) &&
		existing.WalletVersion == event.WalletVersion &&
		equalOptionalInt64(existing.ChargeSequence, optionalChargeInt64(event, event.ChargeSequence)) &&
		existing.EventType == event.EventType &&
		existing.BalanceDeltaMicros == event.BalanceDeltaMicros &&
		existing.ReservedDeltaMicros == event.ReservedDeltaMicros &&
		existing.BalanceAfterMicros == event.BalanceAfterMicros &&
		existing.ReservedAfterMicros == event.ReservedAfterMicros &&
		equalOptionalInt64(existing.ChargeAuthorizedAfterMicros, optionalChargeInt64(event, event.ChargeAuthorizedMicros)) &&
		equalOptionalInt64(existing.ChargeConsumedAfterMicros, optionalChargeInt64(event, event.ChargeConsumedMicros)) &&
		equalOptionalInt64(existing.ChargeReservedAfterMicros, optionalChargeInt64(event, event.ChargeReservedMicros)) &&
		equalOptionalString(existing.ChargeStatus, optionalChargeString(event, event.ChargeStatus)) &&
		pgconv.TimestamptzToTime(existing.OccurredAt).Equal(event.OccurredAt)
}

func equalOptionalUUID(left *uuid.UUID, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func equalOptionalInt64(left *int64, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func equalOptionalString(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func optionalChargeInt64(event redisintegration.OCSEvent, value int64) *int64 {
	if event.ChargeID == nil {
		return nil
	}

	return &value
}

func optionalChargeString(event redisintegration.OCSEvent, value string) *string {
	if event.ChargeID == nil {
		return nil
	}

	return &value
}

func validateWalletProjection(wallet sqlc.Wallet, event redisintegration.OCSEvent) error {
	if wallet.OcsVersion == math.MaxInt64 || event.WalletVersion > wallet.OcsVersion+1 {
		return &PersistenceError{
			Outcome: OutcomeVersionGap,
			Cause: fmt.Errorf(
				"OCS wallet version gap for %s: current=%d event=%d",
				event.WalletID,
				wallet.OcsVersion,
				event.WalletVersion,
			),
		}
	}
	if event.WalletVersion <= wallet.OcsVersion {
		return &PersistenceError{
			Outcome: OutcomeIntegrity,
			Cause: fmt.Errorf(
				"stale OCS wallet version for %s: current=%d event=%d",
				event.WalletID,
				wallet.OcsVersion,
				event.WalletVersion,
			),
		}
	}
	if !safeDeltaMatches(wallet.BalanceMicros, event.BalanceDeltaMicros, event.BalanceAfterMicros) {
		return fmt.Errorf("OCS wallet balance projection mismatch")
	}
	if !safeDeltaMatches(wallet.ReservedMicros, event.ReservedDeltaMicros, event.ReservedAfterMicros) {
		return fmt.Errorf("OCS wallet reservation projection mismatch")
	}

	return nil
}

func validateChargeProjection(
	wallet sqlc.Wallet,
	charge sqlc.Charge,
	event redisintegration.OCSEvent,
) error {
	if charge.Currency != wallet.Currency {
		return fmt.Errorf("OCS charge currency does not match wallet")
	}
	if charge.OcsSequence == math.MaxInt64 || event.ChargeSequence > charge.OcsSequence+1 {
		return &PersistenceError{
			Outcome: OutcomeVersionGap,
			Cause: fmt.Errorf(
				"OCS charge sequence gap for %s: current=%d event=%d",
				*event.ChargeID,
				charge.OcsSequence,
				event.ChargeSequence,
			),
		}
	}
	if event.ChargeSequence <= charge.OcsSequence {
		return &PersistenceError{
			Outcome: OutcomeIntegrity,
			Cause: fmt.Errorf(
				"stale OCS charge sequence for %s: current=%d event=%d",
				*event.ChargeID,
				charge.OcsSequence,
				event.ChargeSequence,
			),
		}
	}

	return nil
}

func validateChargeTransition(
	charge sqlc.Charge,
	event redisintegration.OCSEvent,
) error {
	switch event.EventType {
	case "reserve":
		if !safeDeltaMatches(
			charge.AuthorizedMicros,
			event.ReservedDeltaMicros,
			event.ChargeAuthorizedMicros,
		) || !safeDeltaMatches(
			charge.ReservedMicros,
			event.ReservedDeltaMicros,
			event.ChargeReservedMicros,
		) || event.ChargeConsumedMicros != charge.ConsumedMicros {
			return fmt.Errorf("reserve event charge transition is inconsistent")
		}
	case "consume":
		settled := -event.BalanceDeltaMicros
		if event.ChargeAuthorizedMicros != charge.AuthorizedMicros ||
			!safeDeltaMatches(charge.ConsumedMicros, settled, event.ChargeConsumedMicros) ||
			!safeDeltaMatches(charge.ReservedMicros, event.ReservedDeltaMicros, event.ChargeReservedMicros) {
			return fmt.Errorf("consume event charge transition is inconsistent")
		}
	case "release":
		if event.ChargeAuthorizedMicros != charge.AuthorizedMicros ||
			event.ChargeConsumedMicros != charge.ConsumedMicros ||
			!safeDeltaMatches(charge.ReservedMicros, event.ReservedDeltaMicros, event.ChargeReservedMicros) {
			return fmt.Errorf("release event charge transition is inconsistent")
		}
	case "debit":
		settled := -event.BalanceDeltaMicros
		if !safeDeltaMatches(charge.AuthorizedMicros, settled, event.ChargeAuthorizedMicros) ||
			!safeDeltaMatches(charge.ConsumedMicros, settled, event.ChargeConsumedMicros) ||
			event.ChargeReservedMicros != charge.ReservedMicros {
			return fmt.Errorf("debit event charge transition is inconsistent")
		}
	case "finalize":
		if event.ChargeAuthorizedMicros != charge.AuthorizedMicros ||
			event.ChargeConsumedMicros != charge.ConsumedMicros ||
			event.ChargeReservedMicros != 0 ||
			event.ReservedDeltaMicros != -charge.ReservedMicros {
			return fmt.Errorf("finalize event charge transition is inconsistent")
		}
	}

	return nil
}

func terminalTime(event redisintegration.OCSEvent) pgtype.Timestamptz {
	if event.ChargeStatus == "active" {
		return pgtype.Timestamptz{}
	}

	return pgconv.TimeToTimestamptz(event.OccurredAt)
}

func safeDeltaMatches(current int64, delta int64, after int64) bool {
	if delta > 0 && current > math.MaxInt64-delta {
		return false
	}
	if delta < 0 && current < math.MinInt64-delta {
		return false
	}

	return current+delta == after
}

func retryResult(err error) (Result, error) {
	return Result{
			Outcome: OutcomeRetry,
		}, &PersistenceError{
			Outcome: OutcomeRetry,
			Cause:   err,
		}
}

func integrityResult(err error) (Result, error) {
	return Result{
			Outcome: OutcomeIntegrity,
		}, &PersistenceError{
			Outcome: OutcomeIntegrity,
			Cause:   err,
		}
}

func classifyValidationError(err error) (Result, error) {
	var persistenceError *PersistenceError
	if errors.As(err, &persistenceError) {
		return Result{
			Outcome: persistenceError.Outcome,
		}, err
	}

	return integrityResult(err)
}
