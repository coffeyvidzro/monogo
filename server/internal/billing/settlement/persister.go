package settlement

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Persister struct {
	db *pgxpool.Pool
}

func NewPersister(db *pgxpool.Pool) *Persister {
	if db == nil {
		panic("billing settlement: database is required")
	}

	return &Persister{
		db: db,
	}
}

func (p *Persister) Persist(ctx context.Context, event redisintegration.OCSEvent) error {
	tx, err := p.db.BeginTx(
		ctx,
		pgx.TxOptions{
			IsoLevel: pgx.Serializable,
		},
	)
	if err != nil {
		return fmt.Errorf("begin OCS persistence transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	alreadyPersisted, err := operationAlreadyPersisted(ctx, tx, event)
	if err != nil {
		return err
	}
	if alreadyPersisted {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit OCS idempotency transaction: %w", err)
		}

		return nil
	}

	if err := persistEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit OCS event transaction: %w", err)
	}

	return nil
}

type persistedEvent struct {
	WalletID            uuid.UUID
	OrganizationID      uuid.UUID
	ChargeID            *uuid.UUID
	WalletVersion       int64
	ChargeSequence      *int64
	EventType           string
	BalanceDeltaMicros  int64
	ReservedDeltaMicros int64
	BalanceAfterMicros  int64
	ReservedAfterMicros int64
	OccurredAt          time.Time
}

func operationAlreadyPersisted(
	ctx context.Context,
	tx pgx.Tx,
	event redisintegration.OCSEvent,
) (bool, error) {
	var existing persistedEvent
	err := tx.QueryRow(
		ctx,
		`SELECT wallet_id, organization_id, charge_id, wallet_version,
		        charge_sequence, event_type, balance_delta_micros,
		        reserved_delta_micros, balance_after_micros,
		        reserved_after_micros, occurred_at
		   FROM wallet_events
		  WHERE operation_id = $1`,
		event.OperationID,
	).Scan(
		&existing.WalletID,
		&existing.OrganizationID,
		&existing.ChargeID,
		&existing.WalletVersion,
		&existing.ChargeSequence,
		&existing.EventType,
		&existing.BalanceDeltaMicros,
		&existing.ReservedDeltaMicros,
		&existing.BalanceAfterMicros,
		&existing.ReservedAfterMicros,
		&existing.OccurredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read persisted OCS operation: %w", err)
	}

	if !samePersistedEvent(existing, event) {
		return false, fmt.Errorf(
			"OCS operation %s conflicts with its persisted event",
			event.OperationID,
		)
	}

	return true, nil
}

func samePersistedEvent(existing persistedEvent, event redisintegration.OCSEvent) bool {
	return existing.WalletID == event.WalletID &&
		existing.OrganizationID == event.OrganizationID &&
		equalOptionalUUID(existing.ChargeID, event.ChargeID) &&
		existing.WalletVersion == event.WalletVersion &&
		equalOptionalSequence(existing.ChargeSequence, event) &&
		existing.EventType == event.EventType &&
		existing.BalanceDeltaMicros == event.BalanceDeltaMicros &&
		existing.ReservedDeltaMicros == event.ReservedDeltaMicros &&
		existing.BalanceAfterMicros == event.BalanceAfterMicros &&
		existing.ReservedAfterMicros == event.ReservedAfterMicros &&
		existing.OccurredAt.Equal(event.OccurredAt)
}

func equalOptionalUUID(left *uuid.UUID, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func equalOptionalSequence(sequence *int64, event redisintegration.OCSEvent) bool {
	if event.ChargeID == nil {
		return sequence == nil
	}

	return sequence != nil && *sequence == event.ChargeSequence
}

func persistEvent(
	ctx context.Context,
	tx pgx.Tx,
	event redisintegration.OCSEvent,
) error {
	var balanceMicros int64
	var reservedMicros int64
	var walletVersion int64
	err := tx.QueryRow(
		ctx,
		`SELECT balance_micros, reserved_micros, ocs_version
		   FROM wallets
		  WHERE id = $1 AND organization_id = $2
		  FOR UPDATE`,
		event.WalletID,
		event.OrganizationID,
	).Scan(
		&balanceMicros,
		&reservedMicros,
		&walletVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("OCS wallet %s does not exist", event.WalletID)
	}
	if err != nil {
		return fmt.Errorf("lock OCS wallet projection: %w", err)
	}

	if err := validateWalletProjection(
		walletVersion,
		balanceMicros,
		reservedMicros,
		event,
	); err != nil {
		return err
	}
	if event.ChargeID != nil {
		if err := validateChargeProjection(ctx, tx, event); err != nil {
			return err
		}
	}

	eventID := uuid.New()
	var chargeSequence *int64
	if event.ChargeID != nil {
		chargeSequence = &event.ChargeSequence
	}
	_, err = tx.Exec(
		ctx,
		`INSERT INTO wallet_events (
		     id, wallet_id, organization_id, charge_id, operation_id,
		     wallet_version, charge_sequence, event_type,
		     balance_delta_micros, reserved_delta_micros,
		     balance_after_micros, reserved_after_micros, occurred_at
		 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		eventID,
		event.WalletID,
		event.OrganizationID,
		event.ChargeID,
		event.OperationID,
		event.WalletVersion,
		chargeSequence,
		event.EventType,
		event.BalanceDeltaMicros,
		event.ReservedDeltaMicros,
		event.BalanceAfterMicros,
		event.ReservedAfterMicros,
		event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("insert wallet event: %w", err)
	}

	if err := updateProjections(ctx, tx, event); err != nil {
		return err
	}
	if err := insertLedgerEntry(ctx, tx, eventID, event); err != nil {
		return err
	}

	return nil
}

func validateWalletProjection(
	currentVersion int64,
	currentBalance int64,
	currentReserved int64,
	event redisintegration.OCSEvent,
) error {
	expectedVersion := currentVersion + 1
	if currentVersion == math.MaxInt64 || event.WalletVersion != expectedVersion {
		return fmt.Errorf(
			"OCS wallet version gap for %s: current=%d event=%d",
			event.WalletID,
			currentVersion,
			event.WalletVersion,
		)
	}
	if !safeDeltaMatches(currentBalance, event.BalanceDeltaMicros, event.BalanceAfterMicros) {
		return fmt.Errorf("OCS wallet balance projection mismatch")
	}
	if !safeDeltaMatches(currentReserved, event.ReservedDeltaMicros, event.ReservedAfterMicros) {
		return fmt.Errorf("OCS wallet reservation projection mismatch")
	}

	return nil
}

func validateChargeProjection(
	ctx context.Context,
	tx pgx.Tx,
	event redisintegration.OCSEvent,
) error {
	var organizationID uuid.UUID
	var walletID uuid.UUID
	var currentSequence int64
	err := tx.QueryRow(
		ctx,
		`SELECT organization_id, wallet_id
		   FROM charges
		  WHERE id = $1
		  FOR UPDATE`,
		*event.ChargeID,
	).Scan(
		&organizationID,
		&walletID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("OCS charge %s does not exist", *event.ChargeID)
	}
	if err != nil {
		return fmt.Errorf("lock OCS charge projection: %w", err)
	}
	if organizationID != event.OrganizationID || walletID != event.WalletID {
		return fmt.Errorf("OCS charge identity does not match event")
	}
	err = tx.QueryRow(
		ctx,
		`SELECT COALESCE(MAX(charge_sequence), 0)
		   FROM wallet_events
		  WHERE charge_id = $1`,
		*event.ChargeID,
	).Scan(&currentSequence)
	if err != nil {
		return fmt.Errorf("read OCS charge sequence: %w", err)
	}
	if currentSequence == math.MaxInt64 || event.ChargeSequence != currentSequence+1 {
		return fmt.Errorf(
			"OCS charge sequence gap for %s: current=%d event=%d",
			*event.ChargeID,
			currentSequence,
			event.ChargeSequence,
		)
	}

	return nil
}

func updateProjections(
	ctx context.Context,
	tx pgx.Tx,
	event redisintegration.OCSEvent,
) error {
	command, err := tx.Exec(
		ctx,
		`UPDATE wallets
		    SET balance_micros = $1,
		        reserved_micros = $2,
		        ocs_version = $3,
		        updated_at = NOW()
		  WHERE id = $4 AND organization_id = $5 AND ocs_version = $6`,
		event.BalanceAfterMicros,
		event.ReservedAfterMicros,
		event.WalletVersion,
		event.WalletID,
		event.OrganizationID,
		event.WalletVersion-1,
	)
	if err != nil {
		return fmt.Errorf("update OCS wallet projection: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("OCS wallet projection changed concurrently")
	}

	if event.ChargeID == nil {
		return nil
	}

	var closedAt *time.Time
	if event.ChargeStatus != "active" {
		closedAt = &event.OccurredAt
	}
	command, err = tx.Exec(
		ctx,
		`UPDATE charges
		    SET status = $1,
		        authorized_micros = $2,
		        consumed_micros = $3,
		        reserved_micros = $4,
		        closed_at = $5,
		        updated_at = NOW()
		  WHERE id = $6 AND organization_id = $7 AND wallet_id = $8`,
		event.ChargeStatus,
		event.ChargeAuthorizedMicros,
		event.ChargeConsumedMicros,
		event.ChargeReservedMicros,
		closedAt,
		*event.ChargeID,
		event.OrganizationID,
		event.WalletID,
	)
	if err != nil {
		return fmt.Errorf("update OCS charge projection: %w", err)
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("OCS charge projection was not updated")
	}

	return nil
}

func insertLedgerEntry(
	ctx context.Context,
	tx pgx.Tx,
	eventID uuid.UUID,
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

	_, err := tx.Exec(
		ctx,
		`INSERT INTO wallet_ledger_entries (
		     wallet_event_id, wallet_id, organization_id, charge_id,
		     direction, reason, amount_micros, balance_after_micros,
		     occurred_at
		 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		eventID,
		event.WalletID,
		event.OrganizationID,
		event.ChargeID,
		direction,
		event.EventType,
		amountMicros,
		event.BalanceAfterMicros,
		event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("insert OCS ledger entry: %w", err)
	}

	return nil
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
